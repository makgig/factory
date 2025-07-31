package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RequestDecompression распаковывает сжатые HTTP запросы
func RequestDecompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем что запрос сжат
		if !isRequestCompressed(c) {
			c.Next()
			return
		}

		// Распаковываем тело запроса
		decompressedReader, err := createDecompressedReader(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid compressed data"})
			c.Abort()
			return
		}

		// Заменяем тело запроса на распакованное
		c.Request.Body = decompressedReader
		defer decompressedReader.Close()

		c.Next()
	}
}

// isRequestCompressed проверяет сжат ли входящий запрос
func isRequestCompressed(c *gin.Context) bool {
	contentEncoding := c.GetHeader("Content-Encoding")
	return strings.Contains(contentEncoding, "gzip")
}

// createDecompressedReader создает reader для распаковки gzip данных
func createDecompressedReader(body io.ReadCloser) (*decompressReader, error) {
	gzReader, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}

	return &decompressReader{
		originalBody: body,
		gzReader:     gzReader,
	}, nil
}

// decompressReader обертка для распаковки запросов
type decompressReader struct {
	originalBody io.ReadCloser
	gzReader     *gzip.Reader
}

func (d *decompressReader) Read(p []byte) (n int, err error) {
	return d.gzReader.Read(p)
}

func (d *decompressReader) Close() error {
	// Закрываем оба reader'а
	if err := d.gzReader.Close(); err != nil {
		d.originalBody.Close() // пытаемся закрыть и оригинальный
		return err
	}
	return d.originalBody.Close()
}
