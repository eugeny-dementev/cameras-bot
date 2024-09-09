package main

import (
	"log"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	tg "github.com/amarnathcjd/gogram/telegram"
)

type Application struct {
	tgClient        *tg.Client
	tgBot           *gotgbot.Bot
	tgBotDispatcher *ext.Dispatcher
	tgBotUpdater    *ext.Updater
	state           *State
	cameras         Cameras
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

  a.state = &State{}
  a.state.Setup()

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

	a.tgBotUpdater.StartPolling(a.tgBot, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 30,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 30,
			},
		},
	})

	success, err := a.tgBot.SetChatMenuButton(&gotgbot.SetChatMenuButtonOpts{MenuButton: gotgbot.MenuButtonCommands{}})
	if !success || err != nil {
		log.Fatal("failed to set chat menu button", err)
	} else {
		log.Println("set MenuButtonCommands for all chats:", success)
	}

	return nil
}

func (a *Application) Idle() {
	a.tgClient.Idle()
	a.tgBotUpdater.Idle()
}

func (app *Application) AddCommand(name string, handler func(context *HandlerContext) error) {
	app.tgBotDispatcher.AddHandler(handlers.NewCommand(name, func(bot *gotgbot.Bot, ctx *ext.Context) error {
		log.Println("Command is run", name)
		permissions := app.config.GetPermissionsFor(ctx.EffectiveUser.Id)
		if permissions == nil {
			return nil
		}

		log.Println("Command is allowed", name)
		err := handler(&HandlerContext{bot, ctx, app})
		if err != nil {
			return err
		}

		return nil
	}))
}

func (app *Application) AddCallback(callback string, handler func(context *HandlerContext) error) {
	app.tgBotDispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal(callback), func(bot *gotgbot.Bot, ctx *ext.Context) error {
		log.Println("Callback is run", callback)
		permissions := app.config.GetPermissionsFor(ctx.EffectiveUser.Id)
		if permissions == nil {
			return nil
		}

		log.Println("Callback is allowed", callback)
		err := handler(&HandlerContext{bot, ctx, app})
		if err != nil {
			return err
		}

		return nil
	}))
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
