package telegram

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"../db"
	"../spotify"
	"../youtube"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// HandleUpdates получает апдейты из Telegram через длинный поллинг
func HandleUpdates(bot *tgbotapi.BotAPI) {
	// Настройка апдейта через длинный поллинг (можно настроить вебхук, если требуется)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Ошибка получения апдейтов: %v", err)
	}

	for update := range updates {
		// Обработка команд
		if update.Message != nil {
			// Если получена команда /start, отправляем приветственное сообщение
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handleStartCommand(bot, update.Message)
				default:
					sendTextMessage(bot, update.Message.Chat.ID, "Неизвестная команда")
				}
			} else if update.Message.Text != "" {
				// Обработка текстового сообщения (например, URL плейлиста)
				handlePlaylistURL(bot, update.Message)
			}
		} else if update.CallbackQuery != nil {
			// Обработка нажатий inline-кнопок
			handleCallbackQuery(bot, update.CallbackQuery)
		}
	}
}

// WebhookHandler обрабатывает HTTP-запросы, приходящие от Telegram (при использовании вебхуков)
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&update); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}
	// Для простоты, создаём нового бота с токеном из окружения
	bot, err := tgbotapi.NewBotAPIFromEnv()
	if err != nil {
		http.Error(w, "Ошибка инициализации бота", http.StatusInternalServerError)
		return
	}

	// Обрабатываем апдейт так же, как в HandleUpdates
	if update.Message != nil {
		if update.Message.IsCommand() {
			if update.Message.Command() == "start" {
				handleStartCommand(bot, update.Message)
			}
		} else if update.Message.Text != "" {
			handlePlaylistURL(bot, update.Message)
		}
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(bot, update.CallbackQuery)
	}

	w.WriteHeader(http.StatusOK)
}

// handleStartCommand отправляет приветственное сообщение и inline-кнопки выбора источника
func handleStartCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	// Сохраняем сессию пользователя в БД
	db.CreateUserSession(message.Chat.ID)

	// Формируем inline-кнопки для выбора источника
	var buttons [][]tgbotapi.InlineKeyboardButton
	row := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("YouTube Music", "source_youtube"),
		tgbotapi.NewInlineKeyboardButtonData("Spotify", "source_spotify"),
	)
	buttons = append(buttons, row)

	msg := tgbotapi.NewMessage(message.Chat.ID, "Привет! Выберите источник плейлиста:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// handleCallbackQuery обрабатывает нажатия inline-кнопок
func handleCallbackQuery(bot *tgbotapi.BotAPI, cq *tgbotapi.CallbackQuery) {
	// Определяем тип действия по callback data
	data := cq.Data
	chatID := cq.Message.Chat.ID

	// Простейшая маршрутизация: если нажата кнопка выбора источника
	if strings.HasPrefix(data, "source_") {
		source := strings.TrimPrefix(data, "source_")
		// Сохраняем выбранный источник в сессии пользователя
		db.UpdateUserSession(chatID, "source", source)

		// Отправляем сообщение с информацией и инструкциями по отправке URL плейлиста
		response := "Вы выбрали источник: " + source + ". Теперь отправьте мне URL плейлиста для переноса."
		sendTextMessage(bot, chatID, response)
	}
	// Дополнительно можно обрабатывать callback для выбора целевого сервиса, если требуется

	// Подтверждаем получение callback
	callback := tgbotapi.NewCallback(cq.ID, "Принято")
	if _, err := bot.AnswerCallbackQuery(callback); err != nil {
		log.Printf("Ошибка ответа на callback: %v", err)
	}
}

// handlePlaylistURL обрабатывает сообщение с URL плейлиста
func handlePlaylistURL(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	url := message.Text

	// Валидация URL (простейшая проверка)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		sendTextMessage(bot, chatID, "Неверный формат URL. Пожалуйста, отправьте корректный URL.")
		return
	}

	// Получаем данные сессии пользователя (например, выбранный источник)
	session := db.GetUserSession(chatID)
	source := session["source"]

	// Если URL соответствует «понравившемуся» плейлисту (например, содержит слово "liked"), выполняем синхронизацию лайкнутых песен
	if strings.Contains(url, "liked") {
		sendTextMessage(bot, chatID, "Начинается синхронизация плейлиста 'понравившиеся'...")
		// Вызываем функции синхронизации для выбранного источника
		if source == "spotify" {
			err := spotify.SyncLikedTracks(chatID)
			if err != nil {
				sendTextMessage(bot, chatID, "Ошибка синхронизации Spotify: "+err.Error())
				return
			}
		} else if source == "youtube" {
			err := youtube.SyncLikedMusic(chatID)
			if err != nil {
				sendTextMessage(bot, chatID, "Ошибка синхронизации YouTube: "+err.Error())
				return
			}
		}
		sendTextMessage(bot, chatID, "Синхронизация завершена!")
	} else {
		// Если пользователь передал кастомный плейлист, начинаем поиск/создание плейлиста в целевом сервисе
		sendTextMessage(bot, chatID, "Начинается обработка кастомного плейлиста...")
		if source == "spotify" {
			err := spotify.SyncCustomPlaylist(chatID, url)
			if err != nil {
				sendTextMessage(bot, chatID, "Ошибка синхронизации Spotify: "+err.Error())
				return
			}
		} else if source == "youtube" {
			err := youtube.SyncCustomPlaylist(chatID, url)
			if err != nil {
				sendTextMessage(bot, chatID, "Ошибка синхронизации YouTube: "+err.Error())
				return
			}
		}
		sendTextMessage(bot, chatID, "Плейлист успешно синхронизирован!")
	}
}

// sendTextMessage упрощённая функция отправки текстового сообщения
func sendTextMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}
