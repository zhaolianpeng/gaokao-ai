package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
	max  int

	truncated bool
}

func (w *bodyLogWriter) Write(payload []byte) (int, error) {
	w.capture(payload)
	return w.ResponseWriter.Write(payload)
}

func (w *bodyLogWriter) WriteString(value string) (int, error) {
	w.capture([]byte(value))
	return w.ResponseWriter.WriteString(value)
}

func (w *bodyLogWriter) capture(payload []byte) {
	if w.max <= 0 || len(payload) == 0 {
		return
	}
	remaining := w.max - w.body.Len()
	if remaining <= 0 {
		w.truncated = true
		return
	}
	if len(payload) > remaining {
		_, _ = w.body.Write(payload[:remaining])
		w.truncated = true
		return
	}
	_, _ = w.body.Write(payload)
}

func RequestResponseLogger(maxBodyBytes int) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		requestBody, requestTruncated := readRequestBody(c.Request, maxBodyBytes)
		writer := &bodyLogWriter{ResponseWriter: c.Writer, max: maxBodyBytes}
		c.Writer = writer
		c.Next()
		entry := map[string]any{
			"time":              startedAt.Format(time.RFC3339),
			"type":              "http_access",
			"method":            c.Request.Method,
			"path":              c.Request.URL.Path,
			"query":             c.Request.URL.RawQuery,
			"clientIP":          c.ClientIP(),
			"contentType":       c.ContentType(),
			"pathParams":        paramsToMap(c.Params),
			"requestBody":       requestBody,
			"requestTruncated":  requestTruncated,
			"status":            c.Writer.Status(),
			"responseBody":      printableBody(writer.body.Bytes(), writer.truncated),
			"responseTruncated": writer.truncated,
			"latencyMs":         time.Since(startedAt).Milliseconds(),
		}
		if len(c.Errors) > 0 {
			entry["errors"] = c.Errors.String()
		}
		payload, err := json.Marshal(entry)
		if err != nil {
			_, _ = gin.DefaultWriter.Write([]byte(fmt.Sprintf("{\"time\":%q,\"type\":\"http_access\",\"marshalError\":%q}\n", time.Now().Format(time.RFC3339), err.Error())))
			return
		}
		_, _ = gin.DefaultWriter.Write(append(payload, '\n'))
	}
}

func readRequestBody(request *http.Request, maxBodyBytes int) (string, bool) {
	if request == nil || request.Body == nil {
		return "", false
	}
	contentType := strings.ToLower(strings.TrimSpace(request.Header.Get("Content-Type")))
	if strings.Contains(contentType, "multipart/form-data") {
		return fmt.Sprintf("[multipart request omitted contentType=%s]", contentType), false
	}
	payload, truncated := readAndRestoreBody(request, maxBodyBytes)
	return printableBody(payload, truncated), truncated
}

func readAndRestoreBody(request *http.Request, maxBodyBytes int) ([]byte, bool) {
	limit := int64(maxBodyBytes)
	if limit <= 0 {
		limit = 1024 * 1024
	}
	allPayload, err := io.ReadAll(request.Body)
	if err != nil {
		request.Body = io.NopCloser(bytes.NewReader(nil))
		return []byte(fmt.Sprintf("[read request body failed: %v]", err)), false
	}
	request.Body = io.NopCloser(bytes.NewReader(allPayload))
	truncated := int64(len(allPayload)) > limit
	payload := allPayload
	if truncated {
		payload = payload[:limit]
	}
	return payload, truncated
}

func printableBody(payload []byte, truncated bool) string {
	if len(payload) == 0 {
		return ""
	}
	body := string(payload)
	if truncated {
		return body + "...[truncated]"
	}
	return body
}

func paramsToMap(params gin.Params) map[string]string {
	if len(params) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(params))
	for _, item := range params {
		result[item.Key] = item.Value
	}
	return result
}
