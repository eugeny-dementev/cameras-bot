package main

import (
	"bytes"
	"fmt"

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
    ProtectContent: true,
  })
	if err != nil {
		return err
	}

	return nil
}
