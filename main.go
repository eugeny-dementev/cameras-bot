package main

//#cgo LDFLAGS: -L . -lntgcalls -Wl,-rpath=./
import "C"

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sync"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	_ "eugeny-dementev.github.io/cameras-bot/ntgcalls"
)

type CameraConfig struct {
	Tag    string `json:"tag"`
	Name   string `json:"name"`
	Stream string `json:"stream"`
	Image  string `json:"image"`
}

func (c CameraConfig) String() string {
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

var conf = getConfig()

var camerasClients = Cameras{
	clients: make(map[string]*http.Client),
}

func main() {
	app := Application{}
	err := app.Init()
	if err != nil {
		panic(err)
	} else {
		fmt.Println("App started. Config:", app.config)
	}

	app.AddCommand("start", StartCmd)
	app.AddCommand("about", AboutCmd)
	app.AddCommand("all", AllCmd)

	app.Start()

	for _, cameraConf := range conf.Cameras {
		camerasClients.SetupOne(cameraConf.Tag, cameraConf.Image)
	}

	app.Idle()
}

// start introduces the bot.
func start(bot *gotgbot.Bot, ctx *ext.Context, tags []string) error {
	_, err := bot.SendMessage(
		ctx.EffectiveChat.Id,
		fmt.Sprintf("Hello I'm @%s. I give you access to IP cameras", bot.Username),
		&gotgbot.SendMessageOpts{},
	)
	if err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	return nil
}

func about(bot *gotgbot.Bot, ctx *ext.Context, tags []string) error {
	commandRunLog(ctx, "/about", "Started command")

	_, err := ctx.EffectiveChat.SendMessage(
		bot,
		fmt.Sprintf("Bot to stream video from IP security cameras\nAvailable cameras: `%v`", tags),
		&gotgbot.SendMessageOpts{
			DisableNotification: true,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to echo message: %w", err)
	}

	return nil
}

func commandRunLog(ctx *ext.Context, commandName, message string) {
	chatId := ctx.Message.Chat.Id
	username := ctx.Message.From.Username
	// userId := ctx.Message.From.Id
	log.Printf("[%v][ChatId:%v][User:%v] - %v\n", commandName, chatId, username, message)
}

func getPermissions(userId int64, permissions []CameraPermissions) *CameraPermissions {
	for _, perm := range permissions {
		if perm.UserId == userId {
			return &perm
		}
	}

	return nil
}

func all(bot *gotgbot.Bot, ctx *ext.Context, tags []string) error {
	buffersMap := make(map[string][]byte)
	albumMedias := make([]gotgbot.InputMedia, 0)
	wg := sync.WaitGroup{}
	for _, cameraConf := range conf.Cameras {
		if !slices.Contains(tags, cameraConf.Tag) {
			continue
		}

		wg.Add(1)

		go func(tag string) {
			cameraClient, err := camerasClients.Get(tag)
			if err != nil {
				panic(err)
			}

			failedDueTimeout := false

			cameraResponse, err := cameraClient.Get(cameraConf.Image)
			if err != nil {
				fmt.Println("Request error by timeout", err)
				failedDueTimeout = true
			}

			if !failedDueTimeout && cameraResponse.StatusCode == 200 {
				defer cameraResponse.Body.Close()

				fmt.Println("Camera response", tag, cameraResponse.StatusCode)

				data, err := io.ReadAll(cameraResponse.Body)
				if err != nil {
					fmt.Println("failed to read cameraResponse.Body")
					panic(err)
				}

				buffersMap[tag] = data
			}
			wg.Done()
		}(cameraConf.Tag)
	}
	wg.Wait()

	for key, buffer := range buffersMap {
		fmt.Println("Buffer key", key)
		albumMedias = append(albumMedias, &gotgbot.InputMediaPhoto{
			Media: gotgbot.InputFileByReader(fmt.Sprintf("%v.jpeg", key), bytes.NewReader(buffer)),
		})
	}

	_, err := bot.SendMediaGroup(ctx.EffectiveChat.Id, albumMedias, &gotgbot.SendMediaGroupOpts{})
	if err != nil {
		return err
	}

	return nil
}
