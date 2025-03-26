// pkg/telegram/bot.go
package telegram

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Clean1ines/scps/pkg/logging"
	"github.com/Clean1ines/scps/pkg/pubsub"
	"github.com/Clean1ines/scps/pkg/telegram/handler"
	"github.com/Clean1ines/scps/pkg/telegram/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var Bot *tgbotapi.BotAPI
var pubsubClient *pubsub.PubSubClient

func InitBot(token string) error {
	var err error
	Bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}
	Bot.Debug = false
	if logging.DefaultLogger != nil {
		logging.DefaultLogger.StandardLogger(logging.Info).Printf("Bot started: %s", Bot.Self.UserName)
	}
	return nil
}

func SetupHandlers() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", WebhookHandler)
	mux.HandleFunc("/spotify/callback", OAuthCallback("spotify"))
	mux.HandleFunc("/youtube/callback", OAuthCallback("youtube"))
	mux.HandleFunc("/soundcloud/callback", OAuthCallback("soundcloud"))
	return mux
}

func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	pubsubClient := getPubSubClient() // Implement this helper to reuse client
	playlistService := service.NewPlaylistService(pubsubClient)
	authService := service.NewAuthService()
	messageService := service.NewMessageService(Bot, playlistService, authService)

	if update.CallbackQuery != nil {
		handler := handler.NewCallbackHandler(Bot)
		go handler.HandleCallback(update.CallbackQuery)
	} else if update.Message != nil {
		handler := handler.NewMessageHandler(messageService)
		go handler.HandleMessage(update.Message)
	}

	w.WriteHeader(http.StatusOK)
}

func OAuthCallback(serviceType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" || state == "" {
			http.Error(w, "Invalid parameters", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(state, 10, 64)
		if err != nil {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		authSvc := service.NewAuthService()
		if err := authSvc.HandleAuthCallback(serviceType, code, userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("Authorization successful. You can return to Telegram."))
	}
}

func SetWebhook(webhookURL string) error {
	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return err
	}
	_, err = Bot.Request(wh)
	return err
}

func getPubSubClient() *pubsub.PubSubClient {
	return pubsubClient
}

func SetPubSubClient(client *pubsub.PubSubClient) {
	pubsubClient = client
}
