// pkg/logging/logger.go
package logging

import (
	"cloud.google.com/go/logging"
	"context"
)

// Logger – глобальный объект логгера.
var Logger *logging.Logger

// InitCloudLogger инициализирует клиента Cloud Logging для проекта.
func InitCloudLogger(projectID string) (*logging.Logger, error) {
	ctx := context.Background()
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	// Создаем логгер с именем "scps"
	Logger = client.Logger("scps")
	return Logger, nil
}