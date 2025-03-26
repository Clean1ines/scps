package logging

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging"
)

type Logger struct {
	client *logging.Client
	logger *logging.Logger
}

var DefaultLogger *Logger

const (
	Debug    = logging.Debug
	Info     = logging.Info
	Warning  = logging.Warning
	Error    = logging.Error
	Critical = logging.Critical
)

func InitLogger(projectID string) (*Logger, error) {
	ctx := context.Background()
	client, err := logging.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	logger := client.Logger("scps")
	l := &Logger{
		client: client,
		logger: logger,
	}
	DefaultLogger = l
	return l, nil
}

func (l *Logger) Close() error {
	return l.client.Close()
}

func (l *Logger) Flush() error {
	return l.Close()
}

func (l *Logger) StandardLogger(severity logging.Severity) *LogEntry {
	return &LogEntry{
		logger:   l.logger,
		severity: severity,
	}
}

type LogEntry struct {
	logger   *logging.Logger
	severity logging.Severity
}

func (e *LogEntry) Printf(format string, v ...interface{}) {
	e.logger.Log(logging.Entry{
		Severity: e.severity,
		Payload:  fmt.Sprintf(format, v...),
	})
}

func (e *LogEntry) Fatal(v ...interface{}) {
	e.logger.Log(logging.Entry{
		Severity: Critical,
		Payload:  fmt.Sprint(v...),
	})
	panic(fmt.Sprint(v...))
}

func (e *LogEntry) Fatalf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	e.logger.Log(logging.Entry{
		Severity: Critical,
		Payload:  msg,
	})
	panic(msg)
}
