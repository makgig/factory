package repository

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func New() Storage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge обновляет gauge метрику (заменяет значение)
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.gauges[name] = value
}

// UpdateCounter обновляет counter метрику (добавляет к существующему)
func (m *MemStorage) UpdateCounter(name string, value int64) {
	m.counters[name] += value
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
