package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
  tg "github.com/amarnathcjd/gogram/telegram"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConf struct {
	Id   string `json:"id"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

type Config struct {
	AppHash string       `json:"app_hash"`
	Cameras []CameraConf `json:"cameras"`
	AppId   int          `json:"app_id"`
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
