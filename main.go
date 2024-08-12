package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"

	tg "github.com/amarnathcjd/gogram/telegram"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConf struct {
	Id    string `json:"id"`
	Input string `json:"input"`
}

func (c CameraConf) String() string {
	parsedUrl, err := url.Parse(c.Input)
	if err != nil {
		log.Panic("cannot parse provided input URL", err)
	}

	re := regexp.MustCompile("[a-f0-9]")

	parsedUrl.User = url.UserPassword("root", "root")
	parsedUrl.Host = re.ReplaceAllString(parsedUrl.Host, "*")

	return fmt.Sprintf("Id: %v, URL: %v", c.Id, parsedUrl)
}

type Config struct {
	AppHash string       `json:"app_hash"`
	Cameras []CameraConf `json:"cameras"`
	AppId   int          `json:"app_id"`
}

func (c Config) String() string {
	return fmt.Sprintf("AppHash: %v, Cameras: %v", len(c.AppHash), c.Cameras)
}

func main() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	configDir := path.Join(userHomeDir, ".config/cameras-bot")
	fileSystem := os.DirFS(configDir)
	jsonBytes, err := fs.ReadFile(fileSystem, "config.json")
	if err != nil {
		log.Fatal(err)
	}

	conf := Config{}
	err = json.Unmarshal(jsonBytes, &conf)
	if err != nil {
		log.Fatal(err)
	}

	sessionFilePath := path.Join(configDir, "session")
	mtproto, _ := tg.NewClient(tg.ClientConfig{
		AppID:   int32(conf.AppId),
		AppHash: conf.AppHash,
		Session: sessionFilePath,
	})
	err = mtproto.Start()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("CONFIG:", conf)

	mtproto.Idle()
}
