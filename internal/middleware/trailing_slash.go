package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// TrailingSlash нормализует пути, убирая или добавляя trailing slash
func TrailingSlash() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Если путь заканчивается на /, убираем его (кроме корневого "/")
		if len(path) > 1 && strings.HasSuffix(path, "/") {
			newPath := strings.TrimSuffix(path, "/")

			// Перенаправляем внутренне (без редиректа клиенту)
			c.Request.URL.Path = newPath
			c.Next()
			return
		}

		c.Next()
	}
}
