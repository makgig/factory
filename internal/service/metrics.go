package service

import (
	"errors"
	"strconv"

	"github.com/makgig/factory/internal/models"
	"github.com/makgig/factory/internal/repository"
)

// Предопределенные ошибки
var (
	ErrMetricNameRequired  = errors.New("metric name is required")
	ErrInvalidMetricType   = errors.New("invalid metric type")
	ErrInvalidGaugeValue   = errors.New("invalid gauge value")
	ErrInvalidCounterValue = errors.New("invalid counter value")
	ErrMetricNotFound      = errors.New("metric not found")
)

// MetricsService обрабатывает бизнес-логику метрик
type MetricsService struct {
	storage repository.Storage
}

// New создает новый экземпляр сервиса
func New(storage repository.Storage) *MetricsService {
	return &MetricsService{
		storage: storage,
	}
}

// UpdateMetric обрабатывает обновление метрики
func (s *MetricsService) UpdateMetric(metricType, name, value string) error {
	// Проверяем что имя не пустое
	if name == "" {
		return ErrMetricNameRequired
	}

	// Обрабатываем в зависимости от типа
	switch metricType {
	case "gauge":
		// Парсим как float64
		parsedValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return ErrInvalidGaugeValue
		}

		// Сохраняем в хранилище
		s.storage.UpdateGauge(name, parsedValue)
		return nil

	case "counter":
		// Парсим как int64
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return ErrInvalidCounterValue
		}

		// Сохраняем в хранилище
		s.storage.UpdateCounter(name, parsedValue)
		return nil

	default:
		return ErrInvalidMetricType
	}
}

// GetMetric возвращает значение метрики
func (s *MetricsService) GetMetric(metricType, name string) (string, error) {
	// Проверяем что имя не пустое
	if name == "" {
		return "", ErrMetricNameRequired
	}

	switch metricType {
	case "gauge":
		value, exists := s.storage.GetGauge(name)
		if !exists {
			return "", ErrMetricNotFound
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil

	case "counter":
		value, exists := s.storage.GetCounter(name)
		if !exists {
			return "", ErrMetricNotFound
		}
		return strconv.FormatInt(value, 10), nil

	default:
		return "", ErrInvalidMetricType
	}
}

// MetricsData структура для передачи всех метрик
type MetricsData struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

// GetAllMetrics возвращает все метрики
func (s *MetricsService) GetAllMetrics() MetricsData {
	return MetricsData{
		Gauges:   s.storage.GetAllGauges(),
		Counters: s.storage.GetAllCounters(),
	}
}

func (s *MetricsService) UpdateBatch(items []models.Metrics) error {
	if len(items) == 0 {
		return nil
	}
	if bu, ok := s.storage.(repository.BatchUpdater); ok {
		return bu.UpdateBatch(items)
	}
	for _, m := range items {
		switch m.MType {
		case "gauge":
			if m.Value != nil {
				s.storage.UpdateGauge(m.ID, *m.Value)
			}
		case "counter":
			if m.Delta != nil {
				s.storage.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}
	return nil
}
