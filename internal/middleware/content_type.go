package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// JSONContentType проверяет Content-Type для JSON эндпоинтов
func JSONContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		contentType := c.GetHeader("Content-Type")
		if contentType != "application/json" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type must be application/json"})
			c.Abort()
			return
		}
		c.Next()
	}
}
