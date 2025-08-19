package middleware

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"menlo.ai/jan-api-gateway/app/utils/contextkeys"
)

type BodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w BodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b) // capture response
	return w.ResponseWriter.Write(b)
}

func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Generate and set request ID
		requestID := uuid.New().String()
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, contextkeys.RequestId{}, requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Writer.Header().Set("X-Request-ID", requestID)

		// Read request body
		var reqBody []byte
		if c.Request.Body != nil {
			reqBody, _ = io.ReadAll(c.Request.Body)
			// Restore body so Gin can read it again
			c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		}

		// Wrap writer only if not streaming
		var blw = &BodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()
		contentType := c.Writer.Header().Get("Content-Type")
		isStream := strings.HasPrefix(contentType, "text/event-stream") ||
			strings.HasPrefix(contentType, "application/octet-stream") ||
			strings.HasPrefix(contentType, "application/x-ndjson")

		// Log everything
		duration := time.Since(start)
		responseBody := ""
		if !isStream {
			responseBody = blw.body.String()
		}
		logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"host":       c.Request.Host,
			"path":       c.Request.URL.Path,
			"query":      c.Request.URL.RawQuery,
			"headers":    c.Request.Header,
			"req_body":   string(reqBody),
			"resp_body":  responseBody,
			"latency":    duration.String(),
			"client_ip":  c.ClientIP(),
		}).Info("")
	}
}
