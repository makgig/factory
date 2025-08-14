package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/middleware"
	"github.com/makgig/factory/internal/models"
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
	router.Any("/update/:type/:name/:value", h.updateMetricHandler)

	// JSON endpoint для обновления метрик
	router.POST("/update", middleware.JSONContentType(), h.updateMetricJSONHandler)

	router.POST("/value", middleware.JSONContentType(), h.getMetricJSONHandler)

	// батчевое обновление (строго по ТЗ — со слэшем на конце)
	router.POST("/updates/", middleware.JSONContentType(), h.updateBatch)

	// Роут для получения конкретной метрики - только GET
	router.GET("/value/:type/:name", h.getMetricHandler)

	// Роут для получения всех метрик в HTML - только GET
	router.GET("/", h.getAllMetricsHandler)
}

// getMetricJSONHandler обрабатывает POST /value с JSON телом
func (h *Handler) getMetricJSONHandler(c *gin.Context) {
	var request models.Metrics

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	if request.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric id is required"})
		return
	}

	if request.MType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric type is required"})
		return
	}

	valueStr, err := h.metricsService.GetMetric(request.MType, request.ID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMetricNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Metric not found"})
		case errors.Is(err, service.ErrInvalidMetricType):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metric type"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
		return
	}

	response := models.Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	switch request.MType {
	case "gauge":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse gauge value"})
			return
		}
		response.Value = &value

	case "counter":
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse counter value"})
			return
		}
		response.Delta = &delta
	}

	c.JSON(http.StatusOK, response)
}

// updateMetricJSONHandler обрабатывает POST /update с JSON телом
func (h *Handler) updateMetricJSONHandler(c *gin.Context) {
	var metric models.Metrics

	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	if metric.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric id is required"})
		return
	}

	if metric.MType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "metric type is required"})
		return
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "value is required for gauge metric"})
			return
		}
		if err := h.metricsService.UpdateMetric("gauge", metric.ID, fmt.Sprintf("%g", *metric.Value)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update metric"})
			return
		}

	case "counter":
		if metric.Delta == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "delta is required for counter metric"})
			return
		}
		if err := h.metricsService.UpdateMetric("counter", metric.ID, fmt.Sprintf("%d", *metric.Delta)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update metric"})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// updateMetricHandler обрабатывает POST /update/<ТИП>/<ИМЯ>/<ЗНАЧЕНИЕ>
func (h *Handler) updateMetricHandler(c *gin.Context) {
	if c.Request.Method != http.MethodPost {
		c.String(http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	metricType := c.Param("type")
	metricName := c.Param("name")
	metricValue := c.Param("value")

	if metricName == "" {
		c.String(http.StatusNotFound, "metric name is required")
		return
	}

	if err := h.metricsService.UpdateMetric(metricType, metricName, metricValue); err != nil {
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
	metricType := c.Param("type")
	metricName := c.Param("name")

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
	c.String(http.StatusOK, value)
}

func (h *Handler) updateBatch(c *gin.Context) {
	var items []models.Metrics
	if err := c.ShouldBindJSON(&items); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if len(items) == 0 {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := h.metricsService.UpdateBatch(items); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

// getAllMetricsHandler обрабатывает GET / - возвращает HTML со всеми метриками
func (h *Handler) getAllMetricsHandler(c *gin.Context) {
	metrics := h.metricsService.GetAllMetrics()

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

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
