// pkg/pubsub/dispatcher.go
package pubsub

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"cloud.google.com/go/pubsub"
)

// PubSubClient инкапсулирует клиента, топик и подписку.
type PubSubClient struct {
	Client       *pubsub.Client
	Topic        *pubsub.Topic
	Subscription *pubsub.Subscription
}

// Task представляет задачу для обработки (например, синхронизация плейлиста).
type Task struct {
	UserID      int    `json:"user_id"`
	PlaylistURL string `json:"playlist_url"`
	Service     string `json:"service"` // "spotify", "youtube", "soundcloud"
	Action      string `json:"action"`  // "sync-liked" или "sync-custom"
}

// InitPubSubClient инициализирует клиента Pub/Sub для проекта.
func InitPubSubClient(projectID string) (*PubSubClient, error) {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	topic := client.Topic("playlist-tasks")
	sub := client.Subscription("playlist-tasks-sub")
	return &PubSubClient{
		Client:       client,
		Topic:        topic,
		Subscription: sub,
	}, nil
}

// PublishTask публикует задачу в Pub/Sub.
func (p *PubSubClient) PublishTask(ctx context.Context, task Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	result := p.Topic.Publish(ctx, &pubsub.Message{Data: data})
	_, err = result.Get(ctx)
	return err
}

// StartWorkerPool запускает пул воркеров с заданным числом параллельных задач.
func (p *PubSubClient) StartWorkerPool(workerCount int) {
	ctx := context.Background()
	err := p.Subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var task Task
		if err := json.Unmarshal(msg.Data, &task); err != nil {
			log.Printf("Ошибка разбора задачи: %v", err)
			msg.Nack()
			return
		}
		log.Printf("Начало обработки задачи: userID=%d, service=%s, action=%s", task.UserID, task.Service, task.Action)
		// Здесь должна быть вызвана бизнес-логика обработки задачи.
		// Например, вызов API‑функций синхронизации.
		// Эмуляция обработки:
		time.Sleep(2 * time.Second)
		log.Printf("Задача для userID=%d успешно обработана", task.UserID)
		msg.Ack()
	})
	if err != nil {
		log.Printf("Ошибка получения задач из Pub/Sub: %v", err)
	}
}
