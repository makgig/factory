package repository

import (
	"time"
)

// Storage интерфейс для работы с метриками
type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
	SaveToFile() error
	LoadFromFile() error
	SetStoreConfig(interval time.Duration, syncSave bool)
}
