package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// Close закрывает gzip.Writer и досылает все данные из буфера
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// Gzip возвращает Gin middleware для gzip сжатия
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем Content-Type - сжимаем только JSON и HTML
		shouldCompress := false
		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/json") ||
			strings.Contains(contentType, "text/html") ||
			contentType == "" { // для случаев когда Content-Type устанавливается в handler'е
			shouldCompress = true
		}

		// Проверяем, что клиент умеет получать сжатые данные
		acceptEncoding := c.GetHeader("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		if supportsGzip && shouldCompress {
			// Оборачиваем ResponseWriter для сжатия ответа
			cw := newCompressWriter(c.Writer)
			c.Writer = cw
			defer cw.Close()
		}

		// Проверяем, что клиент отправил сжатые данные
		contentEncoding := c.GetHeader("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")

		if sendsGzip {
			// Оборачиваем тело запроса для распаковки
			cr, err := newCompressReader(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid gzip data"})
				c.Abort()
				return
			}
			c.Request.Body = cr
			defer cr.Close()
		}

		// Передаем управление следующему middleware/handler
		c.Next()
	}
}
