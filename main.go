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
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConf struct {
	Tag   string `json:"tag"`
	Name  string `json:"name"`
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

  return fmt.Sprintf("{Name: %v, Tag: %v, URL: %v}", c.Name, c.Tag, parsedUrl)
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
	bot, err := gotgbot.NewBot(botToken, nil)
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
	dispatcher.AddHandler(handlers.NewCallback(callbackquery.Equal("start_callback"), startCB))

	// Add echo handler to reply to all text messages.
	dispatcher.AddHandler(handlers.NewMessage(message.Text, echo))

	// Start receiving updates.
	err = updater.StartPolling(bot, &ext.PollingOpts{
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
	log.Printf("%s has been started...\n", bot.Username)

	success, err := bot.SetChatMenuButton(&gotgbot.SetChatMenuButtonOpts{MenuButton: gotgbot.MenuButtonCommands{}})
	if !success || err != nil {
		log.Fatal("failed to set chat menu button", err)
	} else {
		log.Println("success:", success)
	}

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
func start(bot *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveMessage.Reply(bot, fmt.Sprintf("Hello, I'm @%s. I <b>repeat</b> all your messages.", bot.User.Username), &gotgbot.SendMessageOpts{
		ParseMode: "html",
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{{
				{Text: "Press me", CallbackData: "start_callback"},
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	success, err := ctx.Message.Chat.SetMenuButton(bot, &gotgbot.SetChatMenuButtonOpts{MenuButton: gotgbot.MenuButtonCommands{}})
	if !success || err != nil {
		log.Fatal("failed to set chat menu button", err)
	} else {
		log.Println("success:", success)
	}

	return nil
}

func startCB(b *gotgbot.Bot, ctx *ext.Context) error {
	cb := ctx.Update.CallbackQuery

	_, err := cb.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
		Text: "You pressed a button!",
	})
	if err != nil {
		return fmt.Errorf("failed to answer start callback query: %w", err)
	}

	_, _, err = cb.Message.EditText(b, "You edited the start message.", nil)
	if err != nil {
		return fmt.Errorf("failed to edit start message text: %w", err)
	}
	return nil
}

func echo(b *gotgbot.Bot, ctx *ext.Context) error {
	_, err := ctx.EffectiveMessage.Reply(b, ctx.EffectiveMessage.Text, nil)
	if err != nil {
		return fmt.Errorf("failed to echo message: %w", err)
	}
	return nil
}
