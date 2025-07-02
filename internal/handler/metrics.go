package handler

import (
	"errors"
	"fmt"
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
	// Роут для обновления метрик - обрабатывает все методы, но разрешает только POST
	updateGroup := router.Group("/update")
	updateGroup.Any("/:type/:name/:value", h.updateMetricHandler)

	// Роут для получения конкретной метрики - только GET
	router.GET("/value/:type/:name", h.getMetricHandler)

	// Роут для получения всех метрик в HTML - только GET
	router.GET("/", h.getAllMetricsHandler)
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
		switch {
		case errors.Is(err, service.ErrMetricNameRequired):
			c.String(http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrInvalidMetricType),
			errors.Is(err, service.ErrInvalidGaugeValue),
			errors.Is(err, service.ErrInvalidCounterValue):
			c.String(http.StatusBadRequest, err.Error())
		default:
			c.String(http.StatusInternalServerError, "Internal server error")
		}
		return
	}
	c.String(http.StatusOK, "OK")
}

// getMetricHandler обрабатывает GET /value/<ТИП>/<ИМЯ>
func (h *Handler) getMetricHandler(c *gin.Context) {
	// Получаем параметры из URL
	metricType := c.Param("type")
	metricName := c.Param("name")

	// Получаем метрику через сервис
	value, err := h.metricsService.GetMetric(metricType, metricName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMetricNotFound):
			c.String(http.StatusNotFound, "Metric not found")
		case errors.Is(err, service.ErrInvalidMetricType):
			c.String(http.StatusBadRequest, "Invalid metric type")
		case errors.Is(err, service.ErrMetricNameRequired):
			c.String(http.StatusBadRequest, "Metric name is required")
		default:
			c.String(http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	// Возвращаем значение метрики
	c.String(http.StatusOK, value)
}

// getAllMetricsHandler обрабатывает GET / - возвращает HTML со всеми метриками
func (h *Handler) getAllMetricsHandler(c *gin.Context) {
	// Получаем все метрики
	metrics := h.metricsService.GetAllMetrics()

	// Генерируем простой HTML
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Metrics</title>
</head>
<body>
    <h1>Metrics</h1>
    <h2>Gauges:</h2>
    <ul>`

	for name, value := range metrics.Gauges {
		html += fmt.Sprintf("<li>%s: %g</li>", name, value)
	}

	html += `</ul>
    <h2>Counters:</h2>
    <ul>`

	for name, value := range metrics.Counters {
		html += fmt.Sprintf("<li>%s: %d</li>", name, value)
	}

	html += `</ul>
</body>
</html>`

	// Отправляем HTML
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
