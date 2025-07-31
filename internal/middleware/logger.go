package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/logger"
	"go.uber.org/zap"
)

// GinLogger — middleware-логер
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Обёртка для подсчета размера ответа
		logWriter := &bodyLogWriter{body: make([]byte, 0), ResponseWriter: c.Writer}
		c.Writer = logWriter

		c.Next()

		duration := time.Since(start)

		// Логируем в правильном порядке
		logger.Log.Info("HTTP request processed",
			zap.String("uri", c.Request.RequestURI),
			zap.String("method", c.Request.Method),
			zap.Duration("duration", duration),
		)

		logger.Log.Info("HTTP response sent",
			zap.Int("status_code", c.Writer.Status()),
			zap.Int("content_size", logWriter.bodySize),
		)
	}
}

// bodyLogWriter — обёртка для подсчета размера ответа
type bodyLogWriter struct {
	gin.ResponseWriter
	body     []byte
	bodySize int
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.bodySize += len(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.bodySize += len(s)
	return w.ResponseWriter.WriteString(s)
}
