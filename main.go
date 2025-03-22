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
	// Инициализация Google Cloud Logging для структурированных логов
	logger, err := logging.InitCloudLogger(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Fatalf("Ошибка инициализации Cloud Logging: %v", err)
	}
	defer logger.Flush()

	// Инициализация Redis для хранения сессий и кэширования API результатов
	storage.InitRedis()

	// Инициализация Telegram-бота с заданным токеном
	telegram.InitBot(os.Getenv("TELEGRAM_BOT_TOKEN"))
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		logger.StandardLogger().Fatal("WEBHOOK_URL не задан")
	}
	if err := telegram.SetWebhook(webhookURL + "/webhook"); err != nil {
		logger.StandardLogger().Fatalf("Ошибка установки вебхука: %v", err)
	}

	// Инициализация Google Cloud Pub/Sub для асинхронной обработки задач
	pubsubClient, err := pubsub.InitPubSubClient(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		logger.StandardLogger().Fatalf("Ошибка инициализации Pub/Sub: %v", err)
	}
	// Запускаем пул воркеров (например, 5 параллельных)
	go pubsubClient.StartWorkerPool(5)

	// Регистрируем HTTP-хэндлеры: для Telegram обновлений и OAuth callback’ов
	http.HandleFunc("/webhook", telegram.WebhookHandler)
	http.HandleFunc("/spotify/callback", telegram.OAuthCallbackHandler("spotify"))
	http.HandleFunc("/youtube/callback", telegram.OAuthCallbackHandler("youtube"))
	http.HandleFunc("/soundcloud/callback", telegram.OAuthCallbackHandler("soundcloud"))

	// Запуск HTTP-сервера с заданными таймаутами (под Cloud Run)
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
