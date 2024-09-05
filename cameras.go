package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/icholy/digest"
)

type Cameras struct {
	clients  map[string]*http.Client
	checkers map[string]*http.Client
	configs  []CameraConfig
}

func (cs *Cameras) Set(tag string, client *http.Client) {
	if cs.clients[tag] == nil {
		cs.clients[tag] = client
	}
}

func (cs *Cameras) Setup(configs []CameraConfig) error {
	cs.configs = configs
	cs.clients = make(map[string]*http.Client)
	cs.checkers = make(map[string]*http.Client)

	for _, conf := range cs.configs {
		if conf.Pass == "" {
			return fmt.Errorf("missing password for camera with tag: %v", conf.Tag)
		}

		client := &http.Client{
			Transport: &digest.Transport{
				Username: conf.User,
				Password: conf.Pass,
			},
			Timeout: time.Second * 2,
		}

		checker := &http.Client{
			Timeout: time.Millisecond * 200,
		}

		cs.checkers[conf.Tag] = checker

		cs.Set(conf.Tag, client)
	}

	return nil
}

func (cs *Cameras) Get(tag string) (*http.Client, error) {
	if cs.clients[tag] == nil {
		return nil, fmt.Errorf("no camera client found for %v", tag)
	}

	return cs.clients[tag], nil
}

func (cs *Cameras) GetAllImages(tags []string) map[string][]byte {
	m := make(map[string][]byte)

	wg := sync.WaitGroup{}
	for _, config := range cs.configs {
		if !slices.Contains(tags, config.Tag) {
			continue
		}

		wg.Add(1)

		go func(tag string) {
			cameraClient, err := cs.Get(tag)
			if err != nil {
				panic(err)
			}

			failedDueTimeout := false

			cameraResponse, err := cameraClient.Get(config.Image())
			if err != nil {
				fmt.Println("Request error by timeout", err)
				failedDueTimeout = true
			}

			if !failedDueTimeout && cameraResponse.StatusCode == 200 {
				defer cameraResponse.Body.Close()

				fmt.Println("Camera response", tag, cameraResponse.StatusCode)

				data, err := io.ReadAll(cameraResponse.Body)
				if err != nil {
					fmt.Println("failed to read cameraResponse.Body")
					panic(err)
				}

				m[tag] = data
			}
			wg.Done()
		}(config.Tag)
	}
	wg.Wait()

	return m
}

func (cs *Cameras) CheckAvailableCameras() (map[string]bool, error) {
	cameraStatuses := make(map[string]bool)
	wg := sync.WaitGroup{}
	for _, config := range cs.configs {
		checker := cs.checkers[config.Tag]
		if checker == nil {
			continue
		}

		wg.Add(1)
		go func() {
			res, err := checker.Get(config.Image())
			if err != nil || os.IsTimeout(err) || res == nil {
				cameraStatuses[config.Tag] = false
				wg.Done()
				return
			}

			if res.StatusCode == 401 {
				cameraStatuses[config.Tag] = true
				wg.Done()
				return
			}
		}()
	}
	wg.Wait()

	return cameraStatuses, nil
}
