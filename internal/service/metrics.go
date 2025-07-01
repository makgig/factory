package service

import (
	"errors"
	"strconv"

	"github.com/makgig/factory/internal/repository"
)

// MetricsService обрабатывает бизнес-логику метрик
type MetricsService struct {
	storage *repository.MemStorage
}

// New создает новый экземпляр сервиса
func New(storage *repository.MemStorage) *MetricsService {
	return &MetricsService{
		storage: storage,
	}
}

// UpdateMetric обрабатывает обновление метрики
func (s *MetricsService) UpdateMetric(metricType, name, value string) error {
	// Проверяем что имя не пустое
	if name == "" {
		return errors.New("metric name is required")
	}

	// Обрабатываем в зависимости от типа
	switch metricType {
	case "gauge":
		// Парсим как float64
		parsedValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("invalid gauge value")
		}

		// Сохраняем в хранилище
		s.storage.UpdateGauge(name, parsedValue)
		return nil

	case "counter":
		// Парсим как int64
		parsedValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("invalid counter value")
		}

		// Сохраняем в хранилище
		s.storage.UpdateCounter(name, parsedValue)
		return nil

	default:
		return errors.New("invalid metric type")
	}
}
