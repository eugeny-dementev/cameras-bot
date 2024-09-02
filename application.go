package main

import (
	"log"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	tg "github.com/amarnathcjd/gogram/telegram"
)

type Application struct {
	cameras         Cameras
	tgClient        *tg.Client
	tgBot           *gotgbot.Bot
	tgBotDispatcher *ext.Dispatcher
	tgBotUpdater    *ext.Updater
	config          Config
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

	a.initTgBotDispather()

	return nil
}

func (a *Application) Start() error {
	err := a.tgClient.Start()
	if err != nil {
		return err
	}

	err = a.initTgBot()
	if err != nil {
		return err
	}

	return nil
}

func (a *Application) Idle() {
	a.tgClient.Idle()
	a.tgBotUpdater.Idle()
}

func (a *Application) initTgClient() error {
	sessionFilePath, err := a.config.GetSessionPath()
	if err != nil {
		return err
	}

	mtproto, _ := tg.NewClient(tg.ClientConfig{
		AppID:   a.config.AppId,
		AppHash: a.config.AppHash,
		Session: sessionFilePath,
	})

	a.tgClient = mtproto

	return nil
}

func (a *Application) initTgBot() error {
	bot, err := gotgbot.NewBot(a.config.BotToken, nil)
	if err != nil {
		return err
	}

	a.tgBot = bot

	return nil
}

func (a *Application) initTgBotDispather() {
	a.tgBotDispatcher = ext.NewDispatcher(&ext.DispatcherOpts{
		// If an error is returned by a handler, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})

	a.tgBotUpdater = ext.NewUpdater(a.tgBotDispatcher, nil)
}
