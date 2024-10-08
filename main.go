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
	app.AddCommand("call", CallCmd)
	app.AddCommand("record", RecordCmd)

	for _, cameraConfig := range app.config.Cameras {
		callback := prepareCallbackHood(cameraConfig.Tag)
		app.AddCallback(callback, RecordTagCallbackFactory(cameraConfig))
	}

	for _, timeRange := range TimeRanges {
		app.AddCallback(prepareCallbackHood(timeRange), RecordTimeCallbackFactory(timeRange))
	}

	err = app.Start()
	if err != nil {
		panic(err)
	} else {
		fmt.Println("App started")
	}

	app.Idle()
}
