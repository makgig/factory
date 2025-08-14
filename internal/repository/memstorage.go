package repository

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/makgig/factory/internal/models"
)

type MemStorage struct {
	gauges        map[string]float64
	counters      map[string]int64
	filePath      string
	storeInterval time.Duration
	syncSave      bool
	stopChan      chan struct{}
	mu            sync.RWMutex
}

// NewWithFile создает MemStorage с поддержкой файла
func New(filePath string) Storage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
		filePath: filePath,
		stopChan: make(chan struct{}),
	}
}

// SetStoreConfig настраивает параметры сохранения
func (m *MemStorage) SetStoreConfig(interval time.Duration, syncSave bool) {
	m.storeInterval = interval
	m.syncSave = syncSave
}

// UpdateGauge обновляет gauge метрику (заменяет значение)
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	m.gauges[name] = value
	m.mu.Unlock()

	if m.syncSave && m.filePath != "" {
		if err := m.SaveToFile(); err != nil {
			log.Printf("Error saving gauge to file: %v", err)
		}
	}
}

// UpdateCounter обновляет counter метрику (добавляет к существующему)
func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.mu.Lock()
	m.counters[name] += value
	m.mu.Unlock()

	if m.syncSave && m.filePath != "" {
		if err := m.SaveToFile(); err != nil {
			log.Printf("Error saving counter to file: %v", err)
		}
	}
}

// StartSavingLoop запускает фоновую горутину для периодического сохранения.
// если storeInterval > 0 и filePath не пустой.
func (m *MemStorage) StartSavingLoop() {
	if m.filePath == "" || m.storeInterval == 0 {
		return
	}

	ticker := time.NewTicker(m.storeInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := m.SaveToFile(); err != nil {
					log.Printf("Error periodic saving to file: %v", err)
				}
			case <-m.stopChan:
				log.Println("Stopping saving loop.")
				return
			}
		}
	}()
}

// StopSavingLoop останавливает фоновую горутину сохранения.
// Должен быть вызван при штатном завершении сервера.
func (m *MemStorage) StopSavingLoop() {
	if m.stopChan != nil {
		close(m.stopChan)
	}
}

// GetGauge возвращает значение gauge метрики
func (m *MemStorage) GetGauge(name string) (float64, bool) {
	m.mu.RLock()
	value, exists := m.gauges[name]
	m.mu.RUnlock()
	return value, exists
}

// GetCounter возвращает значение counter метрики
func (m *MemStorage) GetCounter(name string) (int64, bool) {
	m.mu.RLock()
	value, exists := m.counters[name]
	m.mu.RUnlock()
	return value, exists
}

// GetAllGauges возвращает копию всех gauge метрик
func (m *MemStorage) GetAllGauges() map[string]float64 {
	m.mu.RLock()
	result := make(map[string]float64)
	for name, value := range m.gauges {
		result[name] = value
	}
	m.mu.RUnlock()
	return result
}

// GetAllCounters возвращает копию всех counter метрик
func (m *MemStorage) GetAllCounters() map[string]int64 {
	m.mu.RLock()
	result := make(map[string]int64)
	for name, value := range m.counters {
		result[name] = value
	}
	m.mu.RUnlock()
	return result
}

// SaveToFile сохраняет метрики в JSON файл
func (m *MemStorage) SaveToFile() error {
	if m.filePath == "" {
		return nil
	}

	m.mu.RLock()
	var metrics []models.Metrics

	// Добавляем gauge метрики
	for name, value := range m.gauges {
		v := value // копия для указателя
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}

	// Добавляем counter метрики
	for name, value := range m.counters {
		v := value // копия для указателя
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: "counter",
			Delta: &v,
		})
	}
	m.mu.RUnlock()

	// Сериализуем в JSON
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}

	// Записываем в файл
	err = os.WriteFile(m.filePath, data, 0644)

	return err
}

// LoadFromFile загружает метрики из JSON файла
func (m *MemStorage) LoadFromFile() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Если filePath пуст, ничего не загружаем.
	// Хранилище должно быть очищено в любом случае, чтобы отразить отсутствие загруженных данных.
	m.gauges = make(map[string]float64)
	m.counters = make(map[string]int64)

	if m.filePath == "" {
		return nil
	}

	// Читаем файл
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Если файл пуст, также считаем, что загружать нечего.
	if len(data) == 0 {
		return nil
	}

	// Парсим JSON
	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		// Если JSON некорректен, возвращаем ошибку.
		// Хранилище остается очищенным, что соответствует неудачной загрузке.
		return err
	}

	// Загружаем из файла
	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		case "counter":
			if metric.Delta != nil {
				m.counters[metric.ID] = *metric.Delta
			}
		}
	}

	return nil
}

func (m *MemStorage) UpdateBatch(items []models.Metrics) error {
	if len(items) == 0 {
		return nil
	}

	m.mu.Lock()
	for _, metric := range items {
		switch metric.MType {
		case "gauge":
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		case "counter":
			if metric.Delta != nil {
				m.counters[metric.ID] += *metric.Delta
			}
		}
	}
	m.mu.Unlock()

	if m.syncSave && m.filePath != "" {
		return m.SaveToFile()
	}
	return nil
}
