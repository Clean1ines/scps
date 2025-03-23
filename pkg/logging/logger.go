// pkg/logging/logger.go
package logging

import (
    "context"
    "fmt"
    "log"
    "os"

    "cloud.google.com/go/logging"
)

// Logger представляет обертку для Google Cloud Logging.
type Logger struct {
    client  *logging.Client
    logger  *logging.Logger
    context context.Context
}

// NewLogger создает новый экземпляр Logger, используя переменную окружения GOOGLE_CLOUD_PROJECT.
func NewLogger(ctx context.Context) (*Logger, error) {
    projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
    if projectID == "" {
        projectID = "default-project"
    }
    client, err := logging.NewClient(ctx, projectID)
    if err != nil {
        return nil, err
    }
    l := client.Logger("scps")
    return &Logger{
        client:  client,
        logger:  l,
        context: ctx,
    }, nil
}

// Infof записывает информационное сообщение.
func (l *Logger) Infof(format string, args ...interface{}) {
    entry := logging.Entry{
        Severity: logging.Info,
        Payload:  fmt.Sprintf(format, args...),
    }
    l.logger.Log(entry)
    log.Printf("[INFO] "+format, args...)
}

// Errorf записывает сообщение об ошибке.
func (l *Logger) Errorf(format string, args ...interface{}) {
    entry := logging.Entry{
        Severity: logging.Error,
        Payload:  fmt.Sprintf(format, args...),
    }
    l.logger.Log(entry)
    log.Printf("[ERROR] "+format, args...)
}

// Close закрывает клиента логирования.
func (l *Logger) Close() {
    l.client.Close()
}