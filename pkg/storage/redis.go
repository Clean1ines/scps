// pkg/storage/redis.go
package storage

import (
    "context"
    "time"

    "github.com/go-redis/redis/v8"
)

// NewRedisClient создает клиент Redis и проверяет соединение.
func NewRedisClient(address string) (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:     address,
        Password: "",
        DB:       0,
    })
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := client.Ping(ctx).Err(); err != nil {
        return nil, err
    }
    return client, nil
}