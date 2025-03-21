package main

import (
	"log"
	"net/http"
	"os"
	"time"

	// Импортируем наши локальные модули
	"./db"
	"./oauth"
	"./telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

// Инициализация бота и веб-сервера для обработки Telegram апдейтов и OAuth редиректов.
func main() {
	// Загружаем переменные окружения из файла .env
	err := godotenv.Load()
	if err != nil {
		log.Println("Ошибка загрузки .env файла, продолжаем с системными переменными")
	}

	// Инициализируем соединение с базой данных для хранения сессий
	err = db.InitDB(os.Getenv("DB_CONNECTION_STRING"))
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}
	defer db.CloseDB()

	// Инициализируем Telegram-бота с токеном
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Ошибка инициализации Telegram-бота: %v", err)
	}
	bot.Debug = true // В режиме отладки выводим подробную информацию

	log.Printf("Запущен Telegram бот: %s", bot.Self.UserName)

	// Запускаем горутину для обработки апдейтов Telegram
	go telegram.HandleUpdates(bot)

	// Инициализируем OAuth-конфигурацию для Spotify и YouTube
	oauth.InitOAuthConfig()

	// Запускаем HTTP-сервер для обработки входящих вебхуков и OAuth редиректов
	http.HandleFunc("/spotifyOAuth", oauth.SpotifyOAuthHandler)
	http.HandleFunc("/telegramWebhook", telegram.WebhookHandler)

	// Указываем порт, который будет использовать Firebase Cloud Functions (или Cloud Run)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Значение по умолчанию
	}

	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	log.Printf("HTTP сервер запущен на порту %s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Ошибка запуска HTTP сервера: %v", err)
	}
}
