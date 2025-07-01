package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/service"
)

// Handler содержит зависимости для обработчиков
type Handler struct {
	metricsService *service.MetricsService
}

// New создает новый handler с зависимостями
func New(metricsService *service.MetricsService) *Handler {
	return &Handler{
		metricsService: metricsService,
	}
}

// SetupRoutes настраивает все маршруты
func (h *Handler) SetupRoutes(router *gin.Engine) {
	updateGroup := router.Group("/update")
	updateGroup.Any("/:type/:name/:value", h.updateMetricHandler)
}

// updateMetricHandler обрабатывает POST /update/<ТИП>/<ИМЯ>/<ЗНАЧЕНИЕ>
func (h *Handler) updateMetricHandler(c *gin.Context) {
	// Проверяем метод - только POST разрешен
	if c.Request.Method != http.MethodPost {
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	// Получаем параметры из URL
	metricType := c.Param("type")
	metricName := c.Param("name")
	metricValue := c.Param("value")

	// Проверяем что имя метрики не пустое
	if metricName == "" {
		c.String(http.StatusNotFound, "metric name is required")
		return
	}

	// Вызываем сервис для обработки
	if err := h.metricsService.UpdateMetric(metricType, metricName, metricValue); err != nil {
		// Определяем HTTP статус по тексту ошибки
		switch err.Error() {
		case "metric name is required":
			c.String(http.StatusNotFound, err.Error())
		case "invalid metric type", "invalid gauge value", "invalid counter value":
			c.String(http.StatusBadRequest, err.Error())
		default:
			c.String(http.StatusInternalServerError, "Internal server error")
		}
		return
	}
	c.String(http.StatusOK, "OK")
}
