// pkg/storage/redis.go
package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var Client *redis.Client

// InitRedis инициализирует подключение к Redis.
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
	log.Println("Redis подключен")
	return nil
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
