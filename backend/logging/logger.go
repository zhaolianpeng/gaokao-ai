package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type HourlyFileWriter struct {
	dirPath   string
	prefix    string
	mu        sync.Mutex
	currentID string
	file      *os.File
}

func NewHourlyFileWriter(dirPath, prefix string) (*HourlyFileWriter, error) {
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return nil, err
	}
	return &HourlyFileWriter{dirPath: dirPath, prefix: prefix}, nil
}

func (w *HourlyFileWriter) Write(payload []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotate(time.Now()); err != nil {
		return 0, err
	}
	return w.file.Write(payload)
}

func (w *HourlyFileWriter) rotate(now time.Time) error {
	currentID := now.Format("20060102-15")
	if w.file != nil && w.currentID == currentID {
		return nil
	}
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	logFilePath := filepath.Join(w.dirPath, fmt.Sprintf("%s-%s.log", w.prefix, currentID))
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = file
	w.currentID = currentID
	return nil
}

func Setup(dirPath string) (io.Writer, error) {
	fileWriter, err := NewHourlyFileWriter(dirPath, "gaokao-api")
	if err != nil {
		return nil, err
	}
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	gin.DefaultWriter = multiWriter
	gin.DefaultErrorWriter = multiWriter
	return multiWriter, nil
}
