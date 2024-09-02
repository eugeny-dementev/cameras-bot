package main

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

type HandlerContext struct {
	bot *gotgbot.Bot
	ctx *ext.Context
	app *Application
}
