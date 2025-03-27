// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Clean1ines/scps/pkg/logging"
	"github.com/Clean1ines/scps/pkg/pubsub"
	"github.com/Clean1ines/scps/pkg/storage"
	"github.com/Clean1ines/scps/pkg/telegram"
	"github.com/Clean1ines/scps/pkg/telegram/setup"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

var Client *redis.Client

func InitRedis() error {
	addr := os.Getenv("REDIS_ADDRESS")
	if addr == "" {
		return fmt.Errorf("REDIS_ADDRESS environment variable not set")
	}

	Client = redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	log.Println("Redis connected successfully")
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	logger, err := logging.InitLogger(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Flush()

	if err := telegram.InitBot(os.Getenv("TELEGRAM_BOT_TOKEN")); err != nil {
		logger.StandardLogger(logging.Error).Fatal(err)
	}

	pubsubClient, err := pubsub.InitPubSubClient(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	if err != nil {
		logger.StandardLogger(logging.Error).Fatal(err)
	}
	defer pubsubClient.Client.Close()

	telegram.SetPubSubClient(pubsubClient)
	setup.InitServices(pubsubClient)

	// Initialize Redis once
	if err := storage.InitRedis(); err != nil {
		logger.StandardLogger(logging.Error).Fatalf("Failed to initialize Redis: %v", err)
	}

	// Start worker pool
	go pubsubClient.StartWorkerPool(5)

	// Setup webhook
	webhookURL := os.Getenv("WEBHOOK_URL")
	if err := telegram.SetWebhook(webhookURL + "/webhook"); err != nil {
		logger.StandardLogger(logging.Error).Fatal(err)
	}

	// Setup HTTP handlers
	mux := http.NewServeMux()
	setup.SetupHandlers(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.StandardLogger(logging.Info).Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.StandardLogger(logging.Error).Fatal(err)
	}
}
