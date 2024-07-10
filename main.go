package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"fmt"
	"os"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

func main() {
	appId := os.Getenv("APP_ID")
	appHash := os.Getenv("APP_HASH")

	fmt.Println("ENV:", appId, appHash)
}
