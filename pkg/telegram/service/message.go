package service

import (
	"context"
	"fmt"

	"github.com/Clean1ines/scps/pkg/pubsub"
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

func (s *messageServiceImpl) HandleStart(msg *tgbotapi.Message) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("sync lust", "sync_lust"),
			tgbotapi.NewInlineKeyboardButtonData("list step", "list_step"),
		),
	)

	reply := tgbotapi.NewMessage(msg.Chat.ID, "Welcome to Audi O Shinobu - Your stealthy playlist infiltrator")
	reply.ReplyMarkup = keyboard
	s.bot.Send(reply)
}

func (s *messageServiceImpl) SendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	s.bot.Send(msg)
}

func (s *messageServiceImpl) SendErrorMessage(chatID int64, text string) {
	s.SendMessage(chatID, "❌ "+text)
}

func (s *messageServiceImpl) HandleHelp(msg *tgbotapi.Message) {
	helpText := `Available commands:
/start - Start playlist sync
/help - Show this help message

To sync playlists:
1. Choose sync mode
2. Select platforms
3. Follow the instructions`

	s.SendMessage(msg.Chat.ID, helpText)
}

func (s *messageServiceImpl) SendUnknownCommand(chatID int64) {
	s.SendErrorMessage(chatID, "Unknown command. Use /help for available commands")
}

func (s *messageServiceImpl) ProcessPlaylistURL(msg *tgbotapi.Message) {
	userID := msg.From.ID
	chatID := msg.Chat.ID
	playlistURL := msg.Text

	source, err := storage.GetValue(fmt.Sprintf("session:%d:source", userID))
	if err != nil || source == "" {
		s.SendErrorMessage(chatID, "Session not found. Start with /start")
		return
	}

	target, err := storage.GetValue(fmt.Sprintf("session:%d:target", userID))
	if err != nil || target == "" {
		s.SendErrorMessage(chatID, "Session not found. Start with /start")
		return
	}

	ctx := context.Background()
	task := pubsub.Task{
		UserID:        userID,
		ChatID:        chatID,
		SourceService: source,
		TargetService: target,
		PlaylistURL:   playlistURL,
		Action:        "playlist_conversion",
	}

	if err := s.playlistService.pubsubClient.PublishTask(ctx, task); err != nil {
		s.SendErrorMessage(chatID, "Failed to process playlist")
		return
	}

	s.SendMessage(chatID, "🎵 Your playlist is being processed. You'll be notified when it's ready.")
}
