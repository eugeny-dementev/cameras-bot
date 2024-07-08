package main

import (
	"fmt"
	"os"
)

func main() {
	appId := os.Getenv("APP_ID")
	appHash := os.Getenv("APP_HASH")

	fmt.Println("ENV:", appId, appHash)
}
