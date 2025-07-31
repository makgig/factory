package middleware

import (
	"compress/gzip"
	"strings"

	"github.com/gin-gonic/gin"
)

// ResponseCompression сжимает HTTP ответы
func ResponseCompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем что клиент поддерживает gzip
		if !clientSupportsGzip(c) {
			c.Next()
			return
		}

		// Проверяем что контент нужно сжимать
		if !shouldCompressResponse(c) {
			c.Next()
			return
		}

		// Оборачиваем ResponseWriter для сжатия
		cw := newCompressWriter(c.Writer)
		c.Writer = cw
		defer cw.Close()

		c.Next()
	}
}

// clientSupportsGzip проверяет поддержку gzip клиентом
func clientSupportsGzip(c *gin.Context) bool {
	acceptEncoding := c.GetHeader("Accept-Encoding")
	return strings.Contains(acceptEncoding, "gzip")
}

// shouldCompressResponse определяет нужно ли сжимать ответ
func shouldCompressResponse(c *gin.Context) bool {
	contentType := c.GetHeader("Content-Type")

	// Сжимаем JSON и HTML
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html") ||
		contentType == "" // для случаев когда Content-Type устанавливается в handler'е
}

// compressWriter обертка для сжатия ответов
type compressWriter struct {
	gin.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w gin.ResponseWriter) *compressWriter {
	return &compressWriter{
		ResponseWriter: w,
		zw:             gzip.NewWriter(w),
	}
}

func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

func (c *compressWriter) WriteString(s string) (int, error) {
	return c.zw.Write([]byte(s))
}

func (c *compressWriter) WriteHeader(statusCode int) {
	// Устанавливаем Content-Encoding только для успешных ответов
	if statusCode < 300 {
		c.Header().Set("Content-Encoding", "gzip")
	}
	c.ResponseWriter.WriteHeader(statusCode)
}

func (c *compressWriter) Close() error {
	return c.zw.Close()
}
