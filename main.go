package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConfig struct {
	Tag    string `json:"tag"`
	Name   string `json:"name"`
	Stream string `json:"stream"`
	Image  string `json:"image"`
}

func (c CameraConfig) String() string {
	parsedUrl, err := url.Parse(c.Stream)
	if err != nil {
		log.Panic("cannot parse provided input URL", err)
	}

	re := regexp.MustCompile("[a-f0-9]")

	parsedUrl.User = url.UserPassword("root", "root")
	parsedUrl.Host = re.ReplaceAllString(parsedUrl.Host, "*")

	return fmt.Sprintf("{Name: %v, Tag: %v, URL: %v}", c.Name, c.Tag, parsedUrl)
}

type CameraPermissions struct {
	Tags   []string `json:"tags"`
	UserId int64    `json:"user_id"`
}

func (p CameraPermissions) String() string {
	return fmt.Sprintf("{UserId: %v, Tags: %v}", p.UserId, p.Tags)
}

var conf = getConfig()

var camerasClients = Cameras{
	clients: make(map[string]*http.Client),
}

func main() {
	app := Application{}
	err := app.Init()
	if err != nil {
		panic(err)
	} else {
		fmt.Println("App started. Config:", app.config)
	}

	app.AddCommand("start", StartCmd)
	app.AddCommand("about", AboutCmd)
	app.AddCommand("all", AllCmd)

	app.Start()

	for _, cameraConf := range conf.Cameras {
		camerasClients.SetupOne(cameraConf.Tag, cameraConf.Image)
	}

	app.Idle()
}
