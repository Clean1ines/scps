// main.go
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Clean1ines/scps/pkg/logging"
	"github.com/Clean1ines/scps/pkg/pubsub"
	"github.com/Clean1ines/scps/pkg/storage"
	"github.com/Clean1ines/scps/pkg/telegram"
)

func main() {
	// Инициализация Google Cloud Logging для структурированных логов.
	logger, err := logging.InitCloudLogger(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Fatalf("Ошибка инициализации Cloud Logging: %v", err)
	}
	defer logger.Flush()

	// Инициализация подключения к Redis (удалённый сервер, адрес задаётся в переменной REDIS_ADDRESS).
	storage.InitRedis()

	// Инициализация Telegram-бота с использованием токена из переменной окружения.
	telegram.InitBot(os.Getenv("TELEGRAM_BOT_TOKEN"))

	// Установка вебхука. Переменная WEBHOOK_URL должна быть публичным HTTPS URL вашего сервиса.
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		logger.StandardLogger().Fatal("WEBHOOK_URL не задан")
	}
	if err := telegram.SetWebhook(webhookURL + "/webhook"); err != nil {
		logger.StandardLogger().Fatalf("Ошибка установки вебхука: %v", err)
	}

	// Инициализация клиента Google Cloud Pub/Sub для асинхронной обработки задач.
	pubsubClient, err := pubsub.InitPubSubClient(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		logger.StandardLogger().Fatalf("Ошибка инициализации Pub/Sub: %v", err)
	}
	// Запуск пула воркеров (например, 5 одновременных воркеров).
	go pubsubClient.StartWorkerPool(5)

	// Регистрация HTTP-хэндлеров: обновления Telegram и OAuth callback для Spotify, YouTube и SoundCloud.
	http.HandleFunc("/webhook", telegram.WebhookHandler)
	http.HandleFunc("/spotify/callback", telegram.OAuthCallbackHandler("spotify"))
	http.HandleFunc("/youtube/callback", telegram.OAuthCallbackHandler("youtube"))
	http.HandleFunc("/soundcloud/callback", telegram.OAuthCallbackHandler("soundcloud"))

	// Запуск HTTP-сервера. Переменная PORT задаёт порт сервиса (по умолчанию 8080).
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}
	logger.StandardLogger().Printf("Сервер слушает порт %s", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.StandardLogger().Fatalf("Ошибка HTTP-сервера: %v", err)
	}
}
