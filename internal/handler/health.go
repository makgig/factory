package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db DBPinger
}

func NewHealth(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) SetupRoutes(r *gin.Engine) {
	r.GET("/ping", h.ping)
}

func (h *HealthHandler) ping(c *gin.Context) {
	if h.db == nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}
