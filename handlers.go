package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"

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

		// @EXAMPLE: ffmpeg -t "00:00:05" -i "rtsp://admin:password@192.168.88.111:554/ISAPI/Streaming/Channels/101" "./room.mp4"
		cmd := exec.Command("ffmpeg")
		cmd.Args = append(
			cmd.Args,
			"-t", fmt.Sprintf("00:00:%v", timeRange),
			"-i", input,
			filePath,
		)

		fmt.Println("Prepared command", cmd)

		_, err = c.bot.SendMessage(userId, "Recording is started", &gotgbot.SendMessageOpts{})
		if err != nil {
			return err
		}

		_, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to record: %w", err)
		}

		return nil
	}
}

func prepareCallbackHood(tag string) string {
	return fmt.Sprintf("record_callback_%v", tag)
}
