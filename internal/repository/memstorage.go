package repository

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func New() *MemStorage {
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
