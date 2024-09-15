package main

import (
	"fmt"
	"log"
	"time"

	"eugeny-dementev.github.io/cameras-bot/ntgcalls"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	tg "github.com/amarnathcjd/gogram/telegram"
)

type Application struct {
	tgClient        *tg.Client
	tgInputCall     *tg.InputPhoneCall
	ntgClient       *ntgcalls.Client
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

	a.ntgClient = ntgcalls.NTgCalls()

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
		DropPendingUpdates: false,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 29,
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
	a.ntgClient.Free()
}

func (app *Application) AddCommand(name string, handler func(context *HandlerContext) error) {
	app.tgBotDispatcher.AddHandler(handlers.NewCommand(name, func(bot *gotgbot.Bot, ctx *ext.Context) error {
		log.Println("Command is run", name)

		if ctx.EffectiveUser.Id != app.config.AdminId {
			bot.SendMessage(
				app.config.AdminId,
				fmt.Sprintf("@%v run /%v command", ctx.EffectiveUser.Username, name),
				&gotgbot.SendMessageOpts{},
			)
		}

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

		if ctx.EffectiveUser.Id != app.config.AdminId {
			bot.SendMessage(
				app.config.AdminId,
				fmt.Sprintf("@%v run %v callback", ctx.EffectiveUser.Username, callback),
				&gotgbot.SendMessageOpts{},
			)
		}

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

func (a *Application) VideoCall(stream, username string) {
	fmt.Println("Calls:", a.ntgClient.Calls())
	videoInput := fmt.Sprintf("ffmpeg -i %s -loglevel panic -f rawvideo -r 24 -pix_fmt yuv420p -vf scale=1920:1080 pipe:1", stream)

	rawUser, _ := a.tgClient.ResolveUsername(username)
	user := rawUser.(*tg.UserObj)

	dhConfigRaw, _ := a.tgClient.MessagesGetDhConfig(0, 256)
	dhConfig := dhConfigRaw.(*tg.MessagesDhConfigObj)

	gDesc := ntgcalls.MediaDescription{
		Video: &ntgcalls.VideoDescription{
			InputMode: ntgcalls.InputModeShell,
			Input:     videoInput,
			Width:     1920,
			Height:    1080,
			Fps:       24,
		},
	}

	gAHash, _ := a.ntgClient.CreateP2PCall(user.ID, ntgcalls.DhConfig{
		G:      dhConfig.G,
		P:      dhConfig.P,
		Random: dhConfig.Random,
	}, nil, gDesc)

	protocolRaw := a.ntgClient.GetProtocol()
	protocol := &tg.PhoneCallProtocol{
		UdpP2P:          protocolRaw.UdpP2P,
		UdpReflector:    protocolRaw.UdpReflector,
		MinLayer:        protocolRaw.MinLayer,
		MaxLayer:        protocolRaw.MaxLayer,
		LibraryVersions: protocolRaw.Versions,
	}

	_, _ = a.tgClient.PhoneRequestCall(
		&tg.PhoneRequestCallParams{
			Protocol: protocol,
			UserID:   &tg.InputUserObj{UserID: user.ID, AccessHash: user.AccessHash},
			GAHash:   gAHash,
			RandomID: int32(tg.GenRandInt()),
		},
	)

	a.tgClient.AddRawHandler(&tg.UpdatePhoneCall{}, func(m tg.Update, c *tg.Client) error {
		phoneCall := m.(*tg.UpdatePhoneCall).PhoneCall
		switch phoneCall.(type) {
		case *tg.PhoneCallAccepted:
			call := phoneCall.(*tg.PhoneCallAccepted)
			res, _ := a.ntgClient.ExchangeKeys(user.ID, call.GB, 0)
			a.tgInputCall = &tg.InputPhoneCall{
				ID:         call.ID,
				AccessHash: call.AccessHash,
			}
			a.ntgClient.OnSignal(func(chatId int64, signal []byte) {
				_, _ = a.tgClient.PhoneSendSignalingData(a.tgInputCall, signal)
			})
			callConfirmRes, _ := a.tgClient.PhoneConfirmCall(
				a.tgInputCall,
				res.GAOrB,
				res.KeyFingerprint,
				protocol,
			)
			callRes := callConfirmRes.PhoneCall.(*tg.PhoneCallObj)
			rtcServers := make([]ntgcalls.RTCServer, len(callRes.Connections))
			for i, connection := range callRes.Connections {
				switch connection.(type) {
				case *tg.PhoneConnectionWebrtc:
					rtcServer := connection.(*tg.PhoneConnectionWebrtc)
					rtcServers[i] = ntgcalls.RTCServer{
						ID:       rtcServer.ID,
						Ipv4:     rtcServer.Ip,
						Ipv6:     rtcServer.Ipv6,
						Username: rtcServer.Username,
						Password: rtcServer.Password,
						Port:     rtcServer.Port,
						Turn:     rtcServer.Turn,
						Stun:     rtcServer.Stun,
					}
				case *tg.PhoneConnectionObj:
					phoneServer := connection.(*tg.PhoneConnectionObj)
					rtcServers[i] = ntgcalls.RTCServer{
						ID:      phoneServer.ID,
						Ipv4:    phoneServer.Ip,
						Ipv6:    phoneServer.Ipv6,
						Port:    phoneServer.Port,
						Turn:    true,
						Tcp:     phoneServer.Tcp,
						PeerTag: phoneServer.PeerTag,
					}
				}
			}
			_ = a.ntgClient.ConnectP2P(user.ID, rtcServers, callRes.Protocol.LibraryVersions, callRes.P2PAllowed)
		case *tg.PhoneCallDiscarded:
			call := phoneCall.(*tg.PhoneCallDiscarded)
			fmt.Println("PhoneCallDiscarded reason", call.Reason)
			a.tgInputCall = nil
		}
		return nil
	})

	a.tgClient.AddRawHandler(&tg.UpdatePhoneCallSignalingData{}, func(m tg.Update, c *tg.Client) error {
		signalingData := m.(*tg.UpdatePhoneCallSignalingData).Data
		_ = a.ntgClient.SendSignalingData(user.ID, signalingData)
		return nil
	})
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
	bot, err := gotgbot.NewBot(a.config.BotToken, &gotgbot.BotOpts{
		RequestOpts: &gotgbot.RequestOpts{
			Timeout: time.Second * 30,
		},
	})
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
