package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConf struct {
  Id string `json:"id"`
  User string `json:"user"`
  Pass string `json:"pass"`
}

type Config struct {
	AppId string `json:"app_id"`
  AppHash string `json:"app_hash"`
  Cameras []CameraConf `json:"cameras"`
}

func main() {
	appId := os.Getenv("APP_ID")
	appHash := os.Getenv("APP_HASH")

	myConf := Config{}
  bytes := []byte(`{ "app_id": "some string id", "app_hash": "some hash string", "cameras": [{ "user": "admin", "pass": "password", "id": "bedroom"}]}`)
	err := json.Unmarshal(bytes, &myConf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("ENV:", appId, appHash, myConf)
}
