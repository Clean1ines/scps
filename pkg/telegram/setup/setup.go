package setup

import (
	"net/http"

	"github.com/Clean1ines/scps/pkg/pubsub"
	"github.com/Clean1ines/scps/pkg/telegram/handler"
	"github.com/Clean1ines/scps/pkg/telegram/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var Bot *tgbotapi.BotAPI
var authService *service.AuthService
var playlistService *service.PlaylistService

func InitBot(token string) error {
	var err error
	Bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}
	Bot.Debug = false
	return nil
}

func InitServices(client *pubsub.PubSubClient) {
	authService = service.NewAuthService()
	playlistService = service.NewPlaylistService(client)
}

func SetupHandlers() {
	http.HandleFunc("/webhook", handler.WebhookHandler(Bot, playlistService, authService))
	http.HandleFunc("/spotify/callback", handler.OAuthCallback(authService))
	http.HandleFunc("/youtube/callback", handler.OAuthCallback(authService))
	http.HandleFunc("/soundcloud/callback", handler.OAuthCallback(authService))
}
