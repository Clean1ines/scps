// pkg/storage/redis.go
package storage

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var Client *redis.Client

// InitRedis инициализирует подключение к Redis.
func InitRedis() {
	addr := os.Getenv("REDIS_ADDRESS")
	if addr == "" {
		log.Fatal("REDIS_ADDRESS не задан")
	}
	Client = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := Client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Ошибка подключения к Redis: %v", err)
	}
	log.Println("Redis подключен")
}

// SetValue сохраняет значение по ключу с заданным TTL.
func SetValue(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return Client.Set(ctx, key, value, ttl).Err()
}

// GetValue получает значение по ключу.
func GetValue(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return Client.Get(ctx, key).Result()
}

// DelValue удаляет значение по ключу.
func DelValue(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return Client.Del(ctx, key).Err()
}

// Add Transaction support
func Transaction(fn func() error) error {
	ctx := context.Background()
	return Client.Watch(ctx, func(tx *redis.Tx) error {
		return fn()
	})
}
