package logger

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/config"
	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(cfg *config.ServerConfig) error {
	lvl, err := zap.ParseAtomicLevel(cfg.Loglevel)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	zapCfg := zap.NewProductionConfig()
	// устанавливаем уровень
	zapCfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := zapCfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}

// GinLogger — middleware-логер
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Обёртка для подсчета размера ответа
		blw := &bodyLogWriter{body: make([]byte, 0), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		duration := time.Since(start)

		// Логируем в правильном порядке
		Log.Info("HTTP request processed",
			zap.String("uri", c.Request.RequestURI),
			zap.String("method", c.Request.Method),
			zap.Duration("duration", duration),
		)

		Log.Info("HTTP response sent",
			zap.Int("status_code", c.Writer.Status()),
			zap.Int("content_size", blw.body_size),
		)
	}
}

// bodyLogWriter — обёртка для подсчета размера ответа
type bodyLogWriter struct {
	gin.ResponseWriter
	body      []byte
	body_size int
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body_size += len(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body_size += len(s)
	return w.ResponseWriter.WriteString(s)
}
