package logging

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func LogEvent(eventType string, fields map[string]any) {
	entry := map[string]any{
		"time": time.Now().Format(time.RFC3339),
		"type": eventType,
	}
	for key, value := range fields {
		entry[key] = value
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		_, _ = gin.DefaultWriter.Write([]byte(fmt.Sprintf("{\"time\":%q,\"type\":%q,\"marshalError\":%q}\n", time.Now().Format(time.RFC3339), eventType, err.Error())))
		return
	}
	_, _ = gin.DefaultWriter.Write(append(payload, '\n'))
}

func PreviewString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "...[truncated]"
}
