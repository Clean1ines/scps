// pkg/pubsub/pubsub.go
package pubsub

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "cloud.google.com/go/pubsub"
    "github.com/Clean1ines/scps/pkg/logging"
    "github.com/go-redis/redis/v8"
)

// PubSubClient оборачивает клиента Pub/Sub и хранит ссылку на Redis.
type PubSubClient struct {
    client    *pubsub.Client
    topic     *pubsub.Topic
    sub       *pubsub.Subscription
    projectID string
    redis     *redis.Client
}

// NewPubSubClient создает и инициализирует клиента Pub/Sub.
func NewPubSubClient(ctx context.Context, projectID string, redisClient *redis.Client) (*PubSubClient, error) {
    client, err := pubsub.NewClient(ctx, projectID)
    if err != nil {
        return nil, err
    }
    topic := client.Topic("scps_tasks")
    sub := client.Subscription("scps_tasks_sub")
    return &PubSubClient{
        client:    client,
        topic:     topic,
        sub:       sub,
        projectID: projectID,
        redis:     redisClient,
    }, nil
}

// SyncTask описывает задачу синхронизации плейлистов.
type SyncTask struct {
    Type              string `json:"type"`
    SpotifyPlaylistID string `json:"spotify_playlist_id"`
    YouTubePlaylistID string `json:"youtube_playlist_id"`
    ChatID            int64  `json:"chat_id"`
}

// PublishTask публикует задачу в Pub/Sub.
func (p *PubSubClient) PublishTask(ctx context.Context, task SyncTask) error {
    data, err := json.Marshal(task)
    if err != nil {
        return err
    }
    result := p.topic.Publish(ctx, &pubsub.Message{Data: data})
    _, err = result.Get(ctx)
    return err
}

// updateSyncReport обновляет отчет синхронизации в Redis для пользователя.
func updateSyncReport(ctx context.Context, r *redis.Client, chatID int64, success bool, errMsg string) {
    key := fmt.Sprintf("sync_report_%d", chatID)
    var report struct {
        SuccessCount int      `json:"success_count"`
        Errors       []string `json:"errors"`
    }
    data, err := r.Get(ctx, key).Result()
    if err == nil {
        json.Unmarshal([]byte(data), &report)
    }
    if success {
        report.SuccessCount++
    } else {
        report.Errors = append(report.Errors, errMsg)
    }
    newData, _ := json.Marshal(report)
    r.Set(ctx, key, newData, 24*time.Hour)
}