package handler

import (
	"github.com/Clean1ines/scps/pkg/telegram/middleware"
	"github.com/Clean1ines/scps/pkg/telegram/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type MessageHandler struct {
	messageService service.MessageService
}

func NewMessageHandler(ms service.MessageService) *MessageHandler {
	return &MessageHandler{messageService: ms}
}

func (h *MessageHandler) HandleMessage(msg *tgbotapi.Message) {
	if !middleware.RateLimit(msg.From.ID) {
		h.messageService.SendErrorMessage(msg.Chat.ID, "Please wait before sending another request")
		return
	}

	if msg.IsCommand() {
		h.handleCommand(msg)
		return
	}

	if !middleware.CheckAuth(msg.From.ID) {
		h.messageService.SendErrorMessage(msg.Chat.ID, "Please authorize first using /start")
		return
	}

	h.messageService.ProcessPlaylistURL(msg)
}

func (h *MessageHandler) handleCommand(msg *tgbotapi.Message) {
	switch msg.Command() {
	case "start":
		h.messageService.HandleStart(msg)
	case "help":
		h.messageService.HandleHelp(msg)
	default:
		h.messageService.SendUnknownCommand(msg.Chat.ID)
	}
}
