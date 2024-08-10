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

	myConf := Config{}
	err = json.Unmarshal(jsonBytes, &myConf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("CONFIG:", myConf)
}
