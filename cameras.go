package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/icholy/digest"
)

type Cameras struct {
	clients map[string]*http.Client
}

func (cs *Cameras) Set(tag string, client *http.Client) {
	if cs.clients[tag] == nil {
		cs.clients[tag] = client
	}
}

func (cs *Cameras) Setup(confs []CameraConfig) error {
	cs.clients = make(map[string]*http.Client)

	for _, conf := range confs {
		parsedUrl, err := url.Parse(conf.Image)
		if err != nil {
			return err
		}

		password, hasPass := parsedUrl.User.Password()
		if !hasPass {
			return fmt.Errorf("missing password for camera with tag: %v", conf.Tag)
		}

		client := &http.Client{
			Transport: &digest.Transport{
				Username: parsedUrl.User.Username(),
				Password: password,
			},
			Timeout: time.Second,
		}

		cs.Set(conf.Tag, client)
	}

	return nil
}

func (cs *Cameras) SetupOne(tag, imageHttpUrl string) error {
	parsedUrl, err := url.Parse(imageHttpUrl)
	if err != nil {
		return err
	}

	password, hasPass := parsedUrl.User.Password()
	if !hasPass {
		return fmt.Errorf("missing password for camera with tag: %v", tag)
	}

	client := &http.Client{
		Transport: &digest.Transport{
			Username: parsedUrl.User.Username(),
			Password: password,
		},
		Timeout: time.Second,
	}

	cs.Set(tag, client)

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
	for _, cameraConf := range conf.Cameras {
		if !slices.Contains(tags, cameraConf.Tag) {
			continue
		}

		wg.Add(1)

		go func(tag string) {
			cameraClient, err := cs.Get(tag)
			if err != nil {
				panic(err)
			}

			failedDueTimeout := false

			cameraResponse, err := cameraClient.Get(cameraConf.Image)
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
		}(cameraConf.Tag)
	}
	wg.Wait()

	return m
}
