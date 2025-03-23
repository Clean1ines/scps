// go.mod
module github.com/Clean1ines/scps

go 1.18

require (
    cloud.google.com/go/logging v1.8.0       // Клиент Google Cloud Logging
    cloud.google.com/go/pubsub v1.14.0         // Клиент Google Cloud Pub/Sub
    github.com/agnivade/levenshtein v1.2.0      // Для вычисления расстояния Левенштейна (fuzzy matching)
    github.com/go-redis/redis/v8 v8.11.5        // Клиент Redis для хранения токенов, кэша и сессий
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1 // Telegram Bot API
)