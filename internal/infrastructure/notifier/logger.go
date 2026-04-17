package notifier

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/btcthirst/whale-watcher/internal/domain"
)

type FileAlertLogger struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

func NewFileAlertLogger(filePath string) (*FileAlertLogger, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileAlertLogger{
		file:    f,
		encoder: json.NewEncoder(f),
	}, nil
}

func (l *FileAlertLogger) Send(alert *domain.WhaleAlert) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.encoder.Encode(alert)
}

func (l *FileAlertLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}
