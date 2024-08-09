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

type Config struct {
	AppId string `json:"app_id"`
  AppHash string `json:"app_hash"`
}

func main() {
	appId := os.Getenv("APP_ID")
	appHash := os.Getenv("APP_HASH")

	myConf := Config{}
  bytes := []byte(`{ "app_id": "some string id", "app_hash": "some hash string" }`)
	err := json.Unmarshal(bytes, &myConf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("ENV:", appId, appHash, myConf)
}
