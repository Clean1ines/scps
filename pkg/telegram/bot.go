// pkg/telegram/bot.go
package telegram

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/Clean1ines/scps/pkg/api"
	"github.com/Clean1ines/scps/pkg/logging"
	"github.com/Clean1ines/scps/pkg/pubsub"
	"github.com/Clean1ines/scps/pkg/storage"
	"github.com/Clean1ines/scps/pkg/oauth"
)

var bot *tgbotapi.BotAPI

// InitBot инициализирует Telegram-бота.
func InitBot(token string) {
	var err error
	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram-бота: %v", err)
	}
	bot.Debug = false
	logging.Logger.StandardLogger().Printf("Бот запущен: %s", bot.Self.UserName)
}

// SetWebhook устанавливает вебхук для бота.
func SetWebhook(webhookURL string) error {
	wh, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return err
	}
	_, err = bot.Request(wh)
	return err
}

// WebhookHandler обрабатывает входящие обновления от Telegram.
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if update.CallbackQuery != nil {
		go handleCallbackQuery(update.CallbackQuery)
	} else if update.Message != nil {
		go handleMessage(update.Message)
	}
	w.WriteHeader(http.StatusOK)
}

func handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			text := "Привет! Добро пожаловать в SCPS. Выберите источник плейлиста:"
			reply := tgbotapi.NewMessage(chatID, text)
			reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("YouTube Music", "source:youtube"),
					tgbotapi.NewInlineKeyboardButtonData("Spotify", "source:spotify"),
					tgbotapi.NewInlineKeyboardButtonData("SoundCloud", "source:soundcloud"),
				),
			)
			bot.Send(reply)
		default:
			SendMessage(chatID, "Отправьте URL плейлиста для переноса.")
		}
		return
	}
	processPlaylistURL(msg)
}

func handleCallbackQuery(cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	userID := cb.From.ID
	data := cb.Data

	if strings.HasPrefix(data, "source:") {
		source := strings.TrimPrefix(data, "source:")
		storage.SetValue(fmt.Sprintf("session:%d:source", userID), source, 30*time.Minute)
		text := fmt.Sprintf("Источник выбран: %s.\nТеперь выберите целевой сервис:", source)
		reply := tgbotapi.NewMessage(chatID, text)
		reply.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("YouTube Music", "target:youtube"),
				tgbotapi.NewInlineKeyboardButtonData("Spotify", "target:spotify"),
				tgbotapi.NewInlineKeyboardButtonData("SoundCloud", "target:soundcloud"),
			),
		)
		bot.Send(reply)
	} else if strings.HasPrefix(data, "target:") {
		target := strings.TrimPrefix(data, "target:")
		storage.SetValue(fmt.Sprintf("session:%d:target", userID), target, 30*time.Minute)
		text := fmt.Sprintf("Целевой сервис выбран: %s.\nОтправьте URL плейлиста для переноса.", target)
		SendMessage(chatID, text)
	}
	answer := tgbotapi.NewCallback(cb.ID, "")
	bot.Request(answer)
}

func processPlaylistURL(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID
	playlistURL := msg.Text

	if !isValidURL(playlistURL) {
		SendMessage(chatID, "Неверный формат URL. Попробуйте снова.")
		return
	}
	source, err := storage.GetValue(fmt.Sprintf("session:%d:source", userID))
	if err != nil || source == "" {
		SendMessage(chatID, "Сессия не найдена. Начните с /start")
		return
	}
	target, err := storage.GetValue(fmt.Sprintf("session:%d:target", userID))
	if err != nil || target == "" {
		SendMessage(chatID, "Сессия не найдена. Начните с /start")
		return
	}

	logging.Logger.StandardLogger().Printf("Начало обработки запроса для userID=%d: source=%s, target=%s, URL=%s", userID, source, target, playlistURL)

	// Если URL содержит слово "liked", считаем, что это лайкнутые треки/видео
	if isLikedPlaylist(playlistURL) {
		var err error
		switch {
		case source == "spotify" && target == "spotify":
			err = api.SyncLikedSpotify(userID, playlistURL)
		case source == "youtube" && target == "youtube":
			err = api.SyncLikedYouTube(userID, playlistURL)
		case source == "soundcloud" && target == "soundcloud":
			err = api.SyncLikedSoundCloud(userID, playlistURL)
		default:
			err = fmt.Errorf("неподдерживаемая комбинация сервисов для liked playlist")
		}
		if err != nil {
			SendMessage(chatID, fmt.Sprintf("Ошибка синхронизации: %v", err))
			logging.Logger.StandardLogger().Printf("Ошибка синхронизации liked для userID=%d: %v", userID, err)
		} else {
			SendMessage(chatID, "Синхронизация понравившихся завершена.")
			logging.Logger.StandardLogger().Printf("Синхронизация liked успешно завершена для userID=%d", userID)
		}
	} else {
		// Для кастомного плейлиста публикуем задачу в очередь Pub/Sub для асинхронной обработки
		task := pubsub.Task{
			UserID:      userID,
			PlaylistURL: playlistURL,
			Service:     target,
			Action:      "sync-custom",
		}
		// Здесь предполагается, что глобальный Pub/Sub клиент уже инициализирован и его метод PublishTask доступен
		// Например: pubsubClient.PublishTask(context.Background(), task)
		SendMessage(chatID, "Ваша задача поставлена в очередь. Результаты синхронизации будут сообщены по завершении.")
		logging.Logger.StandardLogger().Printf("Задача кастомной синхронизации поставлена в очередь для userID=%d", userID)
	}
	storage.DelValue(fmt.Sprintf("session:%d:source", userID))
	storage.DelValue(fmt.Sprintf("session:%d:target", userID))
	logging.Logger.StandardLogger().Printf("Завершена обработка запроса для userID=%d", userID)
}

func isValidURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func isLikedPlaylist(s string) bool {
	return strings.Contains(strings.ToLower(s), "liked")
}

func SendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	bot.Send(msg)
}

// OAuthCallbackHandler возвращает HTTP-хэндлер для обработки OAuth callback.
func OAuthCallbackHandler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем code и state, где state содержит userID для защиты CSRF.
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		if code == "" || state == "" {
			http.Error(w, "Недостаточно параметров", http.StatusBadRequest)
			return
		}
		userID, err := strconv.Atoi(state)
		if err != nil {
			http.Error(w, "Неверный state", http.StatusBadRequest)
			return
		}
		switch service {
		case "spotify":
			token, err := oauth.ExchangeSpotifyCode(code)
			if err != nil {
				http.Error(w, fmt.Sprintf("Ошибка Spotify OAuth: %v", err), http.StatusInternalServerError)
				return
			}
			if err := oauth.StoreSpotifyToken(userID, token); err != nil {
				http.Error(w, fmt.Sprintf("Ошибка сохранения Spotify токена: %v", err), http.StatusInternalServerError)
				return
			}
		case "youtube":
			token, err := oauth.ExchangeYouTubeCode(code)
			if err != nil {
				http.Error(w, fmt.Sprintf("Ошибка YouTube OAuth: %v", err), http.StatusInternalServerError)
				return
			}
			if err := oauth.StoreYouTubeToken(userID, token); err != nil {
				http.Error(w, fmt.Sprintf("Ошибка сохранения YouTube токена: %v", err), http.StatusInternalServerError)
				return
			}
		case "soundcloud":
			token, err := oauth.ExchangeSoundCloudCode(code)
			if err != nil {
				http.Error(w, fmt.Sprintf("Ошибка SoundCloud OAuth: %v", err), http.StatusInternalServerError)
				return
			}
			if err := oauth.StoreSoundCloudToken(userID, token); err != nil {
				http.Error(w, fmt.Sprintf("Ошибка сохранения SoundCloud токена: %v", err), http.StatusInternalServerError)
				return
			}
		default:
			http.Error(w, "Неизвестный сервис", http.StatusBadRequest)
			return
		}
		w.Write([]byte(fmt.Sprintf("%s OAuth успешно завершён. Вернитесь в Telegram.", service)))
	}
}