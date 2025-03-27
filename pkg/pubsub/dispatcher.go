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

type Client interface {
	PublishTask(ctx context.Context, task Task) error
}

// Task represents a background task for playlist synchronization
type Task struct {
	UserID        int64  `json:"user_id"`
	ChatID        int64  `json:"chat_id"`
	SourceService string `json:"source_service"`
	TargetService string `json:"target_service"`
	PlaylistURL   string `json:"playlist_url"`
	Action        string `json:"action"`
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
		log.Printf("Начало обработки задачи: userID=%d, source=%s, target=%s", task.UserID, task.SourceService, task.TargetService)
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
