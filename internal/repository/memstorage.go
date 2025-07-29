package repository

import (
	"encoding/json"
	"os"
	"time"

	"github.com/makgig/factory/internal/models"
)

type MemStorage struct {
	gauges        map[string]float64
	counters      map[string]int64
	filePath      string        // путь к файлу для сохранения
	storeInterval time.Duration // интервал сохранения
	lastSaveTime  time.Time     // время последнего сохранения
	syncSave      bool          // флаг синхронного сохранения
}

// NewWithFile создает MemStorage с поддержкой файла
func New(filePath string) Storage {
	return &MemStorage{
		gauges:       make(map[string]float64),
		counters:     make(map[string]int64),
		filePath:     filePath,
		lastSaveTime: time.Now(),
	}
}

// SetStoreConfig настраивает параметры сохранения
func (m *MemStorage) SetStoreConfig(interval time.Duration, syncSave bool) {
	m.storeInterval = interval
	m.syncSave = syncSave
}

// UpdateGauge обновляет gauge метрику (заменяет значение)
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.gauges[name] = value
	m.checkAndSave() // проверяем нужно ли сохранить
}

// UpdateCounter обновляет counter метрику (добавляет к существующему)
func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.counters[name] += value
	m.checkAndSave() // проверяем нужно ли сохранить
}

// checkAndSave проверяет и выполняет сохранение по условиям
func (m *MemStorage) checkAndSave() {
	if m.filePath == "" {
		return // файл не настроен
	}

	// Синхронное сохранение
	if m.syncSave {
		m.SaveToFile()
		return
	}

	// Периодическое сохранение - проверяем прошло ли достаточно времени
	if m.storeInterval > 0 && time.Since(m.lastSaveTime) >= m.storeInterval {
		m.SaveToFile()
	}
}

// GetGauge возвращает значение gauge метрики
func (m *MemStorage) GetGauge(name string) (float64, bool) {
	value, exists := m.gauges[name]
	return value, exists
}

// GetCounter возвращает значение counter метрики
func (m *MemStorage) GetCounter(name string) (int64, bool) {
	value, exists := m.counters[name]
	return value, exists
}

// GetAllGauges возвращает копию всех gauge метрик
func (m *MemStorage) GetAllGauges() map[string]float64 {
	result := make(map[string]float64)
	for name, value := range m.gauges {
		result[name] = value
	}
	return result
}

// GetAllCounters возвращает копию всех counter метрик
func (m *MemStorage) GetAllCounters() map[string]int64 {
	result := make(map[string]int64)
	for name, value := range m.counters {
		result[name] = value
	}
	return result
}

// SaveToFile сохраняет метрики в JSON файл
func (m *MemStorage) SaveToFile() error {
	if m.filePath == "" {
		return nil // Файл не настроен
	}

	// Конвертируем в slice models.Metrics
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

	// Сериализуем в JSON
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}

	// Записываем в файл
	err = os.WriteFile(m.filePath, data, 0644)
	if err == nil {
		m.lastSaveTime = time.Now() // обновляем время последнего сохранения
	}
	return err
}

// LoadFromFile загружает метрики из JSON файла
func (m *MemStorage) LoadFromFile() error {
	if m.filePath == "" {
		return nil // Файл не настроен
	}

	// Читаем файл
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует - создаем пустой
			emptyMetrics := []models.Metrics{}
			data, err := json.MarshalIndent(emptyMetrics, "", "  ")
			if err != nil {
				return err
			}

			err = os.WriteFile(m.filePath, data, 0644)
			if err != nil {
				return err
			}

			m.lastSaveTime = time.Now()
			return nil // файл создан, метрик нет
		}
		return err
	}

	// Парсим JSON
	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	// Очищаем текущие данные
	m.gauges = make(map[string]float64)
	m.counters = make(map[string]int64)

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

	m.lastSaveTime = time.Now()
	return nil
}
