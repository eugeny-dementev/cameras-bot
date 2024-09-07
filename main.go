package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"fmt"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

func main() {
	app := Application{}
	err := app.Init()
	if err != nil {
		panic(err)
	} else {
		fmt.Println("App initialized\nConfig:", app.config)
	}

	app.AddCommand("start", StartCmd)
	app.AddCommand("about", AboutCmd)
	app.AddCommand("all", AllCmd)
  app.AddCommand("record", RecordCmd)
  app.AddCallback("record_callback", RecordCallback)

	err = app.Start()
	if err != nil {
		panic(err)
	} else {
		fmt.Println("App started")
	}

	app.Idle()
}
