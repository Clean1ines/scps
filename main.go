// main.go
package main

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/Clean1ines/scps/pkg/health"
    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/Clean1ines/scps/pkg/oauth"
    "github.com/Clean1ines/scps/pkg/pubsub"
    "github.com/Clean1ines/scps/pkg/telegram"
    "github.com/Clean1ines/scps/pkg/storage"
    "github.com/Clean1ines/scps/pkg/sync"
)

// main инициализирует все модули, запускает фоновые процессы и HTTP-сервер.
func main() {
    // Чтение порта из переменной окружения (по умолчанию 8080)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    ctx := context.Background()

    // Инициализация логгера (Google Cloud Logging)
    logger, err := logging.NewLogger(ctx)
    if err != nil {
        log.Fatalf("Ошибка инициализации логгера: %v", err)
    }
    defer logger.Close()

    // Подключение к Redis
    redisAddr := os.Getenv("REDIS_ADDRESS")
    redisClient, err := storage.NewRedisClient(redisAddr)
    if err != nil {
        logger.Errorf("Ошибка подключения к Redis: %v", err)
        log.Fatalf("Ошибка подключения к Redis: %v", err)
    }

    // Инициализация OAuth для Spotify и YouTube
    oauth.InitSpotify(os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"), os.Getenv("SPOTIFY_REDIRECT_URI"), redisClient, logger)
    oauth.InitYouTube(os.Getenv("YOUTUBE_CLIENT_ID"), os.Getenv("YOUTUBE_CLIENT_SECRET"), os.Getenv("YOUTUBE_REDIRECT_URI"), redisClient, logger)

    // Запуск фонового процесса обновления токенов Spotify каждые 5 минут
    go autoRefreshToken(ctx, redisClient, logger)
    // Запуск периодической двусторонней синхронизации (каждые 30 минут)
    go sync.RunPeriodicSync(ctx, redisClient, logger)

    // Инициализация Pub/Sub для асинхронной обработки задач
    psClient, err := pubsub.NewPubSubClient(ctx, os.Getenv("GOOGLE_CLOUD_PROJECT"), redisClient)
    if err != nil {
        logger.Errorf("Ошибка инициализации Pub/Sub: %v", err)
        log.Fatalf("Ошибка инициализации Pub/Sub: %v", err)
    }
    go pubsub.StartWorkers(ctx, psClient, logger, redisClient)

    // Инициализация Telegram-бота для управления синхронизацией
    bot, err := telegram.NewBot(os.Getenv("TELEGRAM_BOT_TOKEN"), redisClient, logger, psClient)
    if err != nil {
        logger.Errorf("Ошибка инициализации Telegram-бота: %v", err)
        log.Fatalf("Ошибка инициализации Telegram-бота: %v", err)
    }
    go bot.Start()

    // Настройка HTTP-сервера: OAuth callback, эндпоинт здоровья, ручной запуск синхронизации
    mux := http.NewServeMux()
    mux.HandleFunc("/spotify/callback", oauth.SpotifyCallbackHandler)
    mux.HandleFunc("/youtube/callback", oauth.YouTubeCallbackHandler)
    mux.HandleFunc("/health", health.HealthHandler)
    mux.HandleFunc("/sync", sync.SyncHandler) // Эндпоинт для ручной синхронизации

    server := &http.Server{
        Addr:         ":" + port,
        Handler:      mux,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
    }
    logger.Infof("Сервер запущен на порту %s", port)
    if err := server.ListenAndServe(); err != nil {
        logger.Errorf("Ошибка сервера: %v", err)
        log.Fatalf("Ошибка сервера: %v", err)
    }
}

// autoRefreshToken обновляет Spotify access_token каждые 5 минут, используя refresh_token.
func autoRefreshToken(ctx context.Context, redisClient *storage.RedisClient, logger *logging.Logger) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    for {
        <-ticker.C
        tokenJSON, err := redisClient.Get(ctx, "spotify_token").Result()
        if err != nil {
            logger.Errorf("autoRefreshToken: ошибка получения токена: %v", err)
            continue
        }
        var tokenData map[string]interface{}
        if err := json.Unmarshal([]byte(tokenJSON), &tokenData); err != nil {
            logger.Errorf("autoRefreshToken: ошибка парсинга токена: %v", err)
            continue
        }
        refreshToken, ok := tokenData["refresh_token"].(string)
        if !ok || refreshToken == "" {
            logger.Errorf("autoRefreshToken: refresh_token отсутствует")
            continue
        }
        newToken, err := oauth.RefreshSpotifyToken(refreshToken)
        if err != nil {
            logger.Errorf("autoRefreshToken: ошибка обновления токена: %v", err)
            continue
        }
        newJSON, _ := json.Marshal(newToken)
        redisClient.Set(ctx, "spotify_token", newJSON, time.Hour)
        logger.Infof("autoRefreshToken: токен обновлен автоматически")
    }
}