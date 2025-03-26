package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/Clean1ines/scps/pkg/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackHandler struct {
	bot *tgbotapi.BotAPI
}

func NewCallbackHandler(bot *tgbotapi.BotAPI) *CallbackHandler {
	return &CallbackHandler{bot: bot}
}

func (h *CallbackHandler) HandleCallback(cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	userID := cb.From.ID
	data := cb.Data

	if strings.HasPrefix(data, "source:") {
		h.handleSourceSelection(chatID, userID, data)
	} else if strings.HasPrefix(data, "target:") {
		h.handleTargetSelection(chatID, userID, data)
	}

	h.bot.Request(tgbotapi.NewCallback(cb.ID, ""))
}

func (h *CallbackHandler) handleSourceSelection(chatID int64, userID int64, data string) {
	source := strings.TrimPrefix(data, "source:")
	storage.SetValue(fmt.Sprintf("session:%d:source", userID), source, 30*time.Minute)

	text := fmt.Sprintf("Источник выбран: %s.\nТеперь выберите целевой сервис:", source)
	reply := tgbotapi.NewMessage(chatID, text)
	reply.ReplyMarkup = getTargetServicesKeyboard()
	h.bot.Send(reply)
}

func (h *CallbackHandler) handleTargetSelection(chatID int64, userID int64, data string) {
	target := strings.TrimPrefix(data, "target:")
	storage.SetValue(fmt.Sprintf("session:%d:target", userID), target, 30*time.Minute)

	text := fmt.Sprintf("Целевой сервис выбран: %s.\nОтправьте URL плейлиста для переноса.", target)
	h.bot.Send(tgbotapi.NewMessage(chatID, text))
}

func getTargetServicesKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("YouTube Music", "target:youtube"),
			tgbotapi.NewInlineKeyboardButtonData("Spotify", "target:spotify"),
			tgbotapi.NewInlineKeyboardButtonData("SoundCloud", "target:soundcloud"),
		),
	)
}
