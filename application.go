package main

import (
	tg "github.com/amarnathcjd/gogram/telegram"
)

type Application struct {
	cameras  Cameras
	tgClient *tg.Client
	config   Config
}

func (a *Application) Init() error {
	a.config = Config{}
	err := a.config.Setup()
	if err != nil {
		return err
	}

	a.cameras = Cameras{}
	err = a.cameras.Setup(a.config.Cameras)
	if err != nil {
		return err
	}

	err = a.initTgClient()
	if err != nil {
		return err
	}

	return nil
}

func (a *Application) initTgClient() error {
	sessionFilePath, err := a.config.GetSessionPath()
	if err != nil {
		return err
	}

	mtproto, _ := tg.NewClient(tg.ClientConfig{
		AppID:   conf.AppId,
		AppHash: conf.AppHash,
		Session: sessionFilePath,
	})

	a.tgClient = mtproto

	a.tgClient.Start()

	return nil
}

  return nil
}
