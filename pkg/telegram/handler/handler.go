package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Clean1ines/scps/pkg/telegram/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func WebhookHandler(bot *tgbotapi.BotAPI, ps *service.PlaylistService, as *service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var update tgbotapi.Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if update.CallbackQuery != nil {
			callbackHandler := NewCallbackHandler(bot)
			go callbackHandler.HandleCallback(update.CallbackQuery)
		} else if update.Message != nil {
			messageHandler := NewMessageHandler(service.NewMessageService(bot, ps, as))
			go messageHandler.HandleMessage(update.Message)
		}

		w.WriteHeader(http.StatusOK)
	}
}

func OAuthCallback(authService *service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		serviceName := r.URL.Path[1:strings.Index(r.URL.Path, "/callback")]

		if code == "" || state == "" {
			http.Error(w, "Invalid parameters", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(state, 10, 64)
		if err != nil {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		if err := authService.HandleAuthCallback(serviceName, code, userID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte("Authorization successful. You can return to Telegram."))
	}
}
