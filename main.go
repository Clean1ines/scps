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
	"github.com/Clean1ines/scps/pkg/telegram/setup"
)

func main() {
	logger, err := logging.InitLogger(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Flush()

	// Инициализация Redis для хранения сессий и кэширования API результатов
	storage.InitRedis()

	// Initialize Telegram bot and services
	if err := setup.InitBot(os.Getenv("TELEGRAM_BOT_TOKEN")); err != nil {
		logger.StandardLogger(logging.Error).Fatal(err)
	}

	// Initialize Pub/Sub
	pubsubClient, err := pubsub.InitPubSubClient(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		logger.StandardLogger(logging.Error).Printf("Ошибка инициализации Pub/Sub: %v", err)
		os.Exit(1)
	}

	// Initialize services and handlers
	setup.InitServices(pubsubClient)
	setup.SetupHandlers()

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
	logger.StandardLogger(logging.Info).Printf("Сервер слушает порт %s", port)
	if err := srv.ListenAndServe(); err != nil {
		logger.StandardLogger(logging.Error).Fatalf("Ошибка HTTP-сервера: %v", err)
	}
}
