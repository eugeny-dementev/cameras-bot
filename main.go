package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConf struct {
	Id    string `json:"id"`
	Input string `json:"input"`
}

func (c CameraConf) String() string {
	parsedUrl, err := url.Parse(c.Input)
	if err != nil {
		log.Panic("cannot parse provided input URL", err)
	}

	re := regexp.MustCompile("[a-f0-9]")

	parsedUrl.User = url.UserPassword("root", "root")
	parsedUrl.Host = re.ReplaceAllString(parsedUrl.Host, "*")

	return fmt.Sprintf("{Id: %v, URL: %v}", c.Id, parsedUrl)
}

type Config struct {
	AppHash string       `json:"app_hash"`
	Cameras []CameraConf `json:"cameras"`
	AppId   int          `json:"app_id"`
}

func (c Config) String() string {
	return fmt.Sprintf("AppHash: %v, Cameras: %v", len(c.AppHash), c.Cameras)
}

func main() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	configDir := path.Join(userHomeDir, ".config/cameras-bot")
	fileSystem := os.DirFS(configDir)
	jsonBytes, err := fs.ReadFile(fileSystem, "config.json")
	if err != nil {
		log.Fatal(err)
	}

	conf := Config{}
	err = json.Unmarshal(jsonBytes, &conf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("CONFIG:", conf)

	sessionFilePath := path.Join(configDir, "session")
	mtproto, _ := tg.NewClient(tg.ClientConfig{
		AppID:   int32(conf.AppId),
		AppHash: conf.AppHash,
		Session: sessionFilePath,
	})
	err = mtproto.Start()
	if err != nil {
		log.Fatal(err)
	}
	// Get botToken from the environment variable
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		panic("TOKEN environment variable is empty")
	}

	// Create bot from environment value.
	b, err := gotgbot.NewBot(botToken, nil)
	if err != nil {
		panic("failed to create new bot: " + err.Error())
	}

	// Create updater and dispatcher.
	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		// If an error is returned by a handler, log it and continue going.
		Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
			log.Println("an error occurred while handling update:", err.Error())
			return ext.DispatcherActionNoop
		},
		MaxRoutines: ext.DefaultMaxRoutines,
	})
	updater := ext.NewUpdater(dispatcher, nil)

	// /start command to introduce the bot
	dispatcher.AddHandler(handlers.NewCommand("start", start))
	// /source command to send the bot source code
	dispatcher.AddHandler(handlers.NewCommand("source", source))

	// Start receiving updates.
	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})
	if err != nil {
		panic("failed to start polling: " + err.Error())
	}
	log.Printf("%s has been started...\n", b.User.Username)

	mtproto.Idle()
	updater.Idle()
}

func source(b *gotgbot.Bot, ctx *ext.Context) error {
	// Sending a file by file handle
	f, err := os.Open("samples/commandBot/main.go")
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}

	m, err := b.SendDocument(ctx.EffectiveChat.Id,
		gotgbot.InputFileByReader("source.go", f),
		&gotgbot.SendDocumentOpts{
			Caption: "Here is my source code, by file handle.",
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: ctx.EffectiveMessage.MessageId,
			},
		})
	if err != nil {
		return fmt.Errorf("failed to send source: %w", err)
	}

	// Or sending a file by file ID
	_, err = b.SendDocument(ctx.EffectiveChat.Id,
		gotgbot.InputFileByID(m.Document.FileId),
		&gotgbot.SendDocumentOpts{
			Caption: "Here is my source code, sent by file id.",
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: ctx.EffectiveMessage.MessageId,
			},
		})
	if err != nil {
		return fmt.Errorf("failed to send source: %w", err)
	}

	return nil
}

// start introduces the bot.
func start(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveMessage.Reply(b, fmt.Sprintf("Hello, I'm @%s. I <b>repeat</b> all your messages.", b.User.Username), &gotgbot.SendMessageOpts{
		ParseMode: "html",
	})
	if err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}
	return nil
}
