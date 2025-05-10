package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

func StartCmd(c *HandlerContext) error {
	_, err := c.bot.SendMessage(
		c.ctx.EffectiveChat.Id,
		fmt.Sprintf("Hello I'm @%s. I give you access to IP cameras", c.bot.Username),
		&gotgbot.SendMessageOpts{},
	)
	if err != nil {
		return fmt.Errorf("failed to send start message: %w", err)
	}

	return nil
}

func AboutCmd(c *HandlerContext) error {
	permissions := c.app.config.GetPermissionsFor(c.ctx.EffectiveUser.Id)
	if permissions == nil {
		_, err := c.ctx.EffectiveChat.SendMessage(
			c.bot,
			"No Available Cameras",
			&gotgbot.SendMessageOpts{
				DisableNotification: true,
			},
		)
		if err != nil {
			return err
		}

		return nil
	}

	_, err := c.ctx.EffectiveChat.SendMessage(
		c.bot,
		fmt.Sprintf("Bot to stream video from IP security cameras\nAvailable cameras: `%v`", permissions.Tags),
		&gotgbot.SendMessageOpts{
			DisableNotification: true,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func AllCmd(c *HandlerContext) error {
	permissions := c.app.config.GetPermissionsFor(c.ctx.EffectiveUser.Id)
	if permissions == nil {
		_, err := c.ctx.EffectiveChat.SendMessage(
			c.bot,
			"No Available Cameras",
			&gotgbot.SendMessageOpts{
				DisableNotification: true,
			},
		)
		if err != nil {
			return err
		}

		return nil
	}

	cameraStatuses, err := c.app.cameras.CheckAvailableCameras()
	if err != nil {
		return err
	}

	fmt.Println("Camera Statuses:", cameraStatuses)

	tags := make([]string, 0)

	for _, tag := range permissions.Tags {
		if cameraStatuses[tag] {
			tags = append(tags, tag)
		}
	}

	imageBuffers := c.app.cameras.GetAllImages(tags)

	albumMedias := make([]gotgbot.InputMedia, 0)
	for key, buffer := range imageBuffers {
		fmt.Println("Buffer key", key)
		albumMedias = append(albumMedias, &gotgbot.InputMediaPhoto{
			Media: gotgbot.InputFileByReader(fmt.Sprintf("%v.jpeg", key), bytes.NewReader(buffer)),
		})
	}

	_, err = c.bot.SendMediaGroup(c.ctx.EffectiveChat.Id, albumMedias, &gotgbot.SendMediaGroupOpts{
		DisableNotification: true,
		ProtectContent:      true,
		RequestOpts: &gotgbot.RequestOpts{
			Timeout: time.Second * 30,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func RecordCmd(c *HandlerContext) error {
	cameraButtons := make([]gotgbot.InlineKeyboardButton, 0)

	for _, cameraConfig := range c.app.config.Cameras {
		cameraButtons = append(cameraButtons, gotgbot.InlineKeyboardButton{
			Text:         cameraConfig.Name,
			CallbackData: prepareCallbackHood(cameraConfig.Tag),
		})
	}

	_, err := c.bot.SendMessage(c.ctx.EffectiveUser.Id, "Choose camera to record", &gotgbot.SendMessageOpts{
		ParseMode: "html",
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{cameraButtons},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send camera buttons: %w", err)
	}

	return nil
}

func RecordTagCallbackFactory(config CameraConfig) func(c *HandlerContext) error {
	return func(c *HandlerContext) error {
		log.Println("Camera chosen", config.Name)

		c.ctx.EffectiveMessage.Delete(c.bot, &gotgbot.DeleteMessageOpts{})

		timeRangeButtons := make([]gotgbot.InlineKeyboardButton, 0)

		for _, timeRange := range TimeRanges {
			timeRangeButtons = append(timeRangeButtons, gotgbot.InlineKeyboardButton{
				Text:         timeRange,
				CallbackData: prepareCallbackHood(timeRange),
			})
		}

		cq := c.ctx.CallbackQuery
		cq.Answer(c.bot, &gotgbot.AnswerCallbackQueryOpts{})

		userId := c.ctx.EffectiveUser.Id
		c.app.state.Set(userId, "record_input_url", config.Stream())

		fmt.Println("Camera chosen for recording:", config.Tag)
		_, err := c.bot.SendMessage(
			c.ctx.EffectiveUser.Id,
			fmt.Sprintf("Chose time range for %v camera recording", config.Name),
			&gotgbot.SendMessageOpts{
				ParseMode: "html",
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: [][]gotgbot.InlineKeyboardButton{timeRangeButtons},
				},
			},
		)
		if err != nil {
			return fmt.Errorf("failed to send record_callback response: %w", err)
		}

		return nil
	}
}

func RecordTimeCallbackFactory(timeRange string) func(c *HandlerContext) error {
	return func(c *HandlerContext) error {
		log.Println("Time range chosen", timeRange)

		userId := c.ctx.EffectiveUser.Id

		cq := c.ctx.CallbackQuery
		cq.Answer(c.bot, &gotgbot.AnswerCallbackQueryOpts{})

		c.ctx.EffectiveMessage.Delete(c.bot, &gotgbot.DeleteMessageOpts{})

		inputValue, ok := c.app.state.Get(userId, "record_input_url")
		if !ok {
			log.Println("No camera input found", inputValue)
			return nil
		}
		input := inputValue.(string)

		filePath, err := c.app.config.GetTmpRecordingPath(userId, input)
		if err != nil {
			return err
		}

		if _, err := os.Stat(filePath); err == nil {
			os.Remove(filePath)
		}

		var timeArg string
		if timeRange == "60" {
			timeArg = "00:01:00"
		} else {
			timeArg = fmt.Sprintf("00:00:%v", timeRange)
		}

		// @EXAMPLE: ffmpeg -t "00:00:05" -i "rtsp://admin:password@192.168.88.111:554/ISAPI/Streaming/Channels/101" "./room.mp4"
		cmd := exec.Command("ffmpeg")
		if c.app.env.isDocker {
			cmd.Args = append(cmd.Args,
				"-rtsp_transport", "tcp",
			)
		}
		cmd.Args = append(
			cmd.Args,
			"-t", timeArg,
			"-i", input,
			filePath,
		)

		fmt.Println("Prepared command", cmd)

		msgRecStarted, err := c.bot.SendMessage(userId, "Recording is started", &gotgbot.SendMessageOpts{})
		if err != nil {
			return err
		}
		defer msgRecStarted.Delete(c.bot, &gotgbot.DeleteMessageOpts{})

		_, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to record: %w", err)
		}

		// ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0
		// ffprobe -v error -select_streams v:0 -show_entries stream=width,height -of csv=p=0 file.mp4
		probeCmd := exec.Command("ffprobe")
		probeCmd.Args = append(
			probeCmd.Args,
			"-v", "error",
			"-select_streams", "v:0",
			"-show_entries",
			"stream=width,height",
			"-of", "csv=p=0",
			filePath,
		)

		fmt.Println("Prepared command", probeCmd)

		// var out bytes.Buffer
		// cmd.Stdout = &out

		probeOutput, err := probeCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to get video resolution probe: %v", err)
		}

		output := strings.TrimSpace(string(probeOutput))
		resolution := strings.Split(output, ",")

		var width int64 = 1920
		var height int64 = 1080

		if len(resolution) == 2 {
			width, err = strconv.ParseInt(resolution[0], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse video width resolution string: %v", err)
			}

			height, err = strconv.ParseInt(resolution[1], 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse video height resolution string: %v", err)
			}
		}

		fmt.Printf("Parse video resolution: %vx%v\n", width, height)

		for step := range 3 {
			file, err := os.Open(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file %w", err)
			}

			buffer, err := io.ReadAll(io.Reader(file))
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			file.Close()

			_, err = c.bot.SendVideo(
				userId,
				gotgbot.InputFileByReader(filepath.Base(filePath), bytes.NewReader(buffer)),
				&gotgbot.SendVideoOpts{
					Width:               width,
					Height:              height,
					DisableNotification: true,
					ProtectContent:      true,
					RequestOpts: &gotgbot.RequestOpts{
						Timeout: time.Second * 30,
					},
				},
			)

			log.Println("ERR", err)

			if err == nil {
				break
			} else if step == 3 {
				return fmt.Errorf("failed to send file %w", err)
			} else {
				log.Println(fmt.Errorf("failed to send file %w", err))
			}
		}

		err = os.Remove(filePath)
		if err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}

		return nil
	}
}

func CallCmd(c *HandlerContext) error {
	cameraConfig := c.app.config.Cameras[0]
	stream := cameraConfig.Stream()

	c.app.VideoCall(stream, fmt.Sprintf("@%v", c.ctx.EffectiveUser.Username))

	return nil
}

func prepareCallbackHood(tag string) string {
	return fmt.Sprintf("record_callback_%v", tag)
}
