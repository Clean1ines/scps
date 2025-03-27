package service

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageService interface {
	HandleStart(msg *tgbotapi.Message)
	HandleHelp(msg *tgbotapi.Message)
	SendUnknownCommand(chatID int64)
	SendErrorMessage(chatID int64, text string)
	SendMessage(chatID int64, text string)
	ProcessPlaylistURL(msg *tgbotapi.Message)
}

type CallbackService interface {
	HandleCallback(cb *tgbotapi.CallbackQuery)
}
