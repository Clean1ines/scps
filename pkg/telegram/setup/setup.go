package setup

import (
	"net/http"

	"github.com/Clean1ines/scps/pkg/pubsub"
	"github.com/Clean1ines/scps/pkg/telegram/handler"
	"github.com/Clean1ines/scps/pkg/telegram/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	Bot             *tgbotapi.BotAPI
	authService     *service.AuthService
	playlistService *service.PlaylistService
)

func InitServices(client *pubsub.PubSubClient) {
	authService = service.NewAuthService()
	playlistService = service.NewPlaylistService(client)
}

func GetAuthService() *service.AuthService {
	return authService
}

func GetPlaylistService() *service.PlaylistService {
	return playlistService
}

func SetupHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/webhook", handler.WebhookHandler(Bot, playlistService, authService))
	mux.HandleFunc("/spotify/callback", handler.OAuthCallback(authService))
	mux.HandleFunc("/youtube/callback", handler.OAuthCallback(authService))
	mux.HandleFunc("/soundcloud/callback", handler.OAuthCallback(authService))

	// Serve WebApp static files
	mux.Handle("/webapp/", http.StripPrefix("/webapp/", http.FileServer(http.Dir("webapp"))))
}
