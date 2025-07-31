package repository

import (
	"time"
)

// MetricStorage интерфейс для работы с метриками в памяти
type MetricStorage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

// FilePersister интерфейс для управления сохранением метрик в файл
type FilePersist interface {
	SaveToFile() error
	LoadFromFile() error
	SetStoreConfig(interval time.Duration, syncSave bool)
}

// BackgroundSaver интерфейс для управления фоновым сохранением
type BackgroundSaver interface {
	StartSavingLoop()
	StopSavingLoop()
}

// интерфейс-комбинация
type Storage interface {
	MetricStorage
	FilePersist
	BackgroundSaver
}
