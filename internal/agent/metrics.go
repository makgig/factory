package agent

// MetricsStorage хранит собранные метрики в агенте
type MetricsStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

// NewMetricsStorage создает новое хранилище метрик
func NewMetricsStorage() *MetricsStorage {
	return &MetricsStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// SetGauge устанавливает значение gauge метрики
func (m *MetricsStorage) SetGauge(name string, value float64) {
	m.gauges[name] = value
}

// AddCounter добавляет значение к counter метрике
func (m *MetricsStorage) AddCounter(name string, value int64) {
	m.counters[name] += value
}

// GetGauge возвращает значение gauge метрики
func (m *MetricsStorage) GetGauge(name string) (float64, bool) {
	value, exists := m.gauges[name]
	return value, exists
}

// GetCounter возвращает значение counter метрики
func (m *MetricsStorage) GetCounter(name string) (int64, bool) {
	value, exists := m.counters[name]
	return value, exists
}

// AllMetrics содержит все метрики для отправки
type AllMetrics struct {
	Gauges   map[string]float64
	Counters map[string]int64
}

// GetAll возвращает копию всех метрик
func (m *MetricsStorage) GetAll() AllMetrics {
	// Создаем копии карт для безопасности
	gaugesCopy := make(map[string]float64)
	for name, value := range m.gauges {
		gaugesCopy[name] = value
	}

	countersCopy := make(map[string]int64)
	for name, value := range m.counters {
		countersCopy[name] = value
	}

	return AllMetrics{
		Gauges:   gaugesCopy,
		Counters: countersCopy,
	}
}
