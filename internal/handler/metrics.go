package handler

import (
	"net/http"
	"strings"

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
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/update/", h.updateMetricHandler)
}

// updateMetricHandler обрабатывает POST /update/<ТИП>/<ИМЯ>/<ЗНАЧЕНИЕ>
func (h *Handler) updateMetricHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим URL: /update/gauge/temperature/23.5
	path := strings.TrimPrefix(r.URL.Path, "/update/")
	parts := strings.Split(path, "/")

	// Проверяем что есть все 3 части: тип/имя/значение
	if len(parts) != 3 {
		http.Error(w, "Invalid URL format", http.StatusNotFound)
		return
	}

	metricType := parts[0]
	metricName := parts[1]
	metricValue := parts[2]

	// Вызываем сервис для обработки
	if err := h.metricsService.UpdateMetric(metricType, metricName, metricValue); err != nil {
		// Определяем HTTP статус по тексту ошибки
		switch err.Error() {
		case "metric name is required":
			http.Error(w, err.Error(), http.StatusNotFound)
		case "invalid metric type", "invalid gauge value", "invalid counter value":
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Успех
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
