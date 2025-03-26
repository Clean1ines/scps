package service

import (
	"fmt"

	"github.com/Clean1ines/scps/pkg/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type messageServiceImpl struct {
	bot             *tgbotapi.BotAPI
	playlistService *PlaylistService
	authService     *AuthService
}

func NewMessageService(bot *tgbotapi.BotAPI, ps *PlaylistService, as *AuthService) MessageService {
	return &messageServiceImpl{
		bot:             bot,
		playlistService: ps,
		authService:     as,
	}
}

func (s *messageServiceImpl) SendErrorMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	s.bot.Send(msg)
}

func (s *messageServiceImpl) HandleStart(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := "Привет! Добро пожаловать в SCPS. Выберите источник плейлиста:"
	reply := tgbotapi.NewMessage(chatID, text)
	reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("YouTube Music", "source:youtube"),
			tgbotapi.NewInlineKeyboardButtonData("Spotify", "source:spotify"),
			tgbotapi.NewInlineKeyboardButtonData("SoundCloud", "source:soundcloud"),
		),
	)
	s.bot.Send(reply)
}

func (s *messageServiceImpl) HandleHelp(msg *tgbotapi.Message) {
	// Implementation
}

func (s *messageServiceImpl) SendUnknownCommand(chatID int64) {
	s.SendErrorMessage(chatID, "Неизвестная команда")
}

func (s *messageServiceImpl) ProcessPlaylistURL(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	playlistURL := msg.Text

	source, err := storage.GetValue(fmt.Sprintf("session:%d:source", userID))
	if err != nil || source == "" {
		s.SendErrorMessage(chatID, "Сессия не найдена. Начните с /start")
		return
	}

	target, err := storage.GetValue(fmt.Sprintf("session:%d:target", userID))
	if err != nil || target == "" {
		s.SendErrorMessage(chatID, "Сессия не найдена. Начните с /start")
		return
	}

	processor := NewPlaylistProcessor()
	if err := processor.ProcessPlaylist(userID, playlistURL, source, target); err != nil {
		s.SendErrorMessage(chatID, fmt.Sprintf("Ошибка: %v", err))
		return
	}

	s.bot.Send(tgbotapi.NewMessage(chatID, "Задача по синхронизации плейлиста поставлена в очередь"))

	// Очищаем сессию
	storage.DelValue(fmt.Sprintf("session:%d:source", userID))
	storage.DelValue(fmt.Sprintf("session:%d:target", userID))
}
