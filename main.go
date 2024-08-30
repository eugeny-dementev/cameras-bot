package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/icholy/digest"

	tg "github.com/amarnathcjd/gogram/telegram"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/callbackquery"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConf struct {
	Tag    string `json:"tag"`
	Name   string `json:"name"`
	Stream string `json:"stream"`
	Image  string `json:"image"`
}

func (c CameraConf) String() string {
	parsedUrl, err := url.Parse(c.Stream)
	if err != nil {
		log.Panic("cannot parse provided input URL", err)
	}

	re := regexp.MustCompile("[a-f0-9]")

	parsedUrl.User = url.UserPassword("root", "root")
	parsedUrl.Host = re.ReplaceAllString(parsedUrl.Host, "*")

	return fmt.Sprintf("{Name: %v, Tag: %v, URL: %v}", c.Name, c.Tag, parsedUrl)
}

type CameraPermissions struct {
	Tags   []string `json:"tags"`
	UserId int64    `json:"user_id"`
}

func (p CameraPermissions) String() string {
	return fmt.Sprintf("{UserId: %v, Tags: %v}", p.UserId, p.Tags)
}

type Config struct {
	AppHash     string              `json:"app_hash"`
	BotToken    string              `json:"bot_token"`
	Cameras     []CameraConf        `json:"cameras"`
	Permissions []CameraPermissions `json:"permissions"`
	AppId       int                 `json:"app_id"`
	AdminId     int64               `json:"admin_id"`
}

func (c Config) String() string {
	return fmt.Sprintf("AppHash: %v\nAdminId: %v\nCameras: %v\nPermissions: %v", len(c.AppHash), c.AdminId, c.Cameras, c.Permissions)
}

func (c Config) GetPermissionsFor(userId int64) *CameraPermissions {
	for _, perm := range c.Permissions {
		if perm.UserId == userId {
			return &perm
		}
	}

	return nil
}

var conf = getConfig()

type Cameras struct {
	clients map[string]*http.Client
}

func (cs *Cameras) Set(tag string, client *http.Client) {
	if cs.clients[tag] == nil {
		cs.clients[tag] = client
	}
}

func (cs *Cameras) Setup(tag, imageHttpUrl string) error {
	parsedUrl, err := url.Parse(imageHttpUrl)
	if err != nil {
		return err
	}

	password, hasPass := parsedUrl.User.Password()
	if !hasPass {
		return fmt.Errorf("missing password for camera with tag: %v", tag)
	}

	client := &http.Client{
		Transport: &digest.Transport{
			Username: parsedUrl.User.Username(),
			Password: password,
		},
		Timeout: time.Second * 2,
	}

	cs.Set(tag, client)

	return nil
}

func (cs *Cameras) Get(tag string) (*http.Client, error) {
	if cs.clients[tag] == nil {
		return nil, fmt.Errorf("no camera client found for %v", tag)
	}

	return cs.clients[tag], nil
}

var camerasClients = Cameras{
	clients: make(map[string]*http.Client),
}

func main() {
	fmt.Println("CONFIG:", conf)

	for _, cameraConf := range conf.Cameras {
		if cameraConf.Tag == "lr" || cameraConf.Tag == "cr" {
			camerasClients.Setup(cameraConf.Tag, cameraConf.Image)
		}
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configDir := path.Join(userHomeDir, ".config/cameras-bot")
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
	botToken := conf.BotToken
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

	// /about command to provide info about bot and what it can
	dispatcher.AddHandler(handlers.NewCommand("about", about))

	// /all command to pull album with pictures immediately from all cameras at once
	dispatcher.AddHandler(handlers.NewCommand("all", all))

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
		log.Println("set MenuButtonCommands for all chats:", success)
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

func about(bot *gotgbot.Bot, ctx *ext.Context) error {
	commandRunLog(ctx, "/about", "Started command")

	_, err := ctx.EffectiveChat.SendMessage(
		bot,
		"Bot to stream video from IP security cameras",
		&gotgbot.SendMessageOpts{
			DisableNotification: true,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to echo message: %w", err)
	}

	if ctx.EffectiveSender.ChatId == conf.AdminId {
		perms := getPermissions(conf.AdminId, conf.Permissions)
		_, err := ctx.EffectiveChat.SendMessage(
			bot,
			fmt.Sprintf("Available cameras: %v", perms.Tags),
			&gotgbot.SendMessageOpts{
				DisableNotification: true,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to echo message: %w", err)
		}
	}

	return nil
}

func commandRunLog(ctx *ext.Context, commandName, message string) {
	chatId := ctx.Message.Chat.Id
	username := ctx.Message.From.Username
	// userId := ctx.Message.From.Id
	log.Printf("[%v][ChatId:%v][User:%v] - %v\n", commandName, chatId, username, message)
}

func getConfig() Config {
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

	return conf
}

func getPermissions(userId int64, permissions []CameraPermissions) *CameraPermissions {
	for _, perm := range permissions {
		if perm.UserId == userId {
			return &perm
		}
	}

	return nil
}

func all(bot *gotgbot.Bot, ctx *ext.Context) error {
	var lrConf *CameraConf
	var crConf *CameraConf
	for _, cameraConf := range conf.Cameras {
		if cameraConf.Tag == "lr" {
			lrConf = &cameraConf
		}
	}
	if lrConf == nil {
		log.Fatal("lrConf not found", lrConf)
	} else {
		fmt.Println("lrConf found", lrConf)
	}

	for _, cameraConf := range conf.Cameras {
		if cameraConf.Tag == "cr" {
			crConf = &cameraConf
		}
	}
	if crConf == nil {
		log.Fatal("crConf not found", crConf)
	} else {
		fmt.Println("crConf found", crConf)
	}

	lrClient, err := camerasClients.Get(lrConf.Tag)
	if err != nil {
		return err
	}

	lrResp, err := lrClient.Get(lrConf.Image)
	if err != nil {
		return err
	}

	fmt.Println("Image lr status code:", lrResp.StatusCode, lrResp.Header)

	defer lrResp.Body.Close()

	crClient, err := camerasClients.Get(crConf.Tag)
	if err != nil {
		return err
	}

	crResp, err := crClient.Get(crConf.Image)
	if err != nil {
		return err
	}

	fmt.Println("Image cr status code:", crResp.StatusCode, crResp.Header)

	defer crResp.Body.Close()

	// m, err := bot.SendPhoto(ctx.EffectiveChat.Id, gotgbot.InputFileByReader("lr.jpeg", lrResp.Body), &gotgbot.SendPhotoOpts{})
	// if err != nil {
	// 	return err
	// }
	// fmt.Println("Reponse from sending a photo", m)

	albumMedias := make([]gotgbot.InputMedia, 0)
	if lrResp.StatusCode >= 200 && lrResp.StatusCode <= 299 {
		albumMedias = append(albumMedias, &gotgbot.InputMediaPhoto{
			Media: gotgbot.InputFileByReader(fmt.Sprintf("%v.jpeg", lrConf.Tag), lrResp.Body),
		})
	}
	if crResp.StatusCode >= 200 && crResp.StatusCode <= 299 {
		albumMedias = append(albumMedias, &gotgbot.InputMediaPhoto{
			Media: gotgbot.InputFileByReader(fmt.Sprintf("%v.jpeg", crConf.Tag), crResp.Body),
		})
	}

	bot.SendMediaGroup(ctx.EffectiveSender.ChatId, []gotgbot.InputMedia(albumMedias), &gotgbot.SendMediaGroupOpts{
		ProtectContent: true,
	})

	return nil
}
