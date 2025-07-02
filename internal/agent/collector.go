package agent

import (
	"math/rand"
	"runtime"
)

// Collector собирает метрики из runtime
type Collector struct {
	metrics   *MetricsStorage
	pollCount int64
}

// NewCollector создает новый коллектор
func NewCollector(metrics *MetricsStorage) *Collector {
	return &Collector{
		metrics: metrics,
	}
}

// CollectMetrics собирает все runtime метрики
func (c *Collector) CollectMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Собираем gauge метрики
	c.metrics.SetGauge("Alloc", float64(m.Alloc))
	c.metrics.SetGauge("BuckHashSys", float64(m.BuckHashSys))
	c.metrics.SetGauge("Frees", float64(m.Frees))
	c.metrics.SetGauge("GCCPUFraction", m.GCCPUFraction)
	c.metrics.SetGauge("GCSys", float64(m.GCSys))
	c.metrics.SetGauge("HeapAlloc", float64(m.HeapAlloc))
	c.metrics.SetGauge("HeapIdle", float64(m.HeapIdle))
	c.metrics.SetGauge("HeapInuse", float64(m.HeapInuse))
	c.metrics.SetGauge("HeapObjects", float64(m.HeapObjects))
	c.metrics.SetGauge("HeapReleased", float64(m.HeapReleased))
	c.metrics.SetGauge("HeapSys", float64(m.HeapSys))
	c.metrics.SetGauge("LastGC", float64(m.LastGC))
	c.metrics.SetGauge("Lookups", float64(m.Lookups))
	c.metrics.SetGauge("MCacheInuse", float64(m.MCacheInuse))
	c.metrics.SetGauge("MCacheSys", float64(m.MCacheSys))
	c.metrics.SetGauge("MSpanInuse", float64(m.MSpanInuse))
	c.metrics.SetGauge("MSpanSys", float64(m.MSpanSys))
	c.metrics.SetGauge("Mallocs", float64(m.Mallocs))
	c.metrics.SetGauge("NextGC", float64(m.NextGC))
	c.metrics.SetGauge("NumForcedGC", float64(m.NumForcedGC))
	c.metrics.SetGauge("NumGC", float64(m.NumGC))
	c.metrics.SetGauge("OtherSys", float64(m.OtherSys))
	c.metrics.SetGauge("PauseTotalNs", float64(m.PauseTotalNs))
	c.metrics.SetGauge("StackInuse", float64(m.StackInuse))
	c.metrics.SetGauge("StackSys", float64(m.StackSys))
	c.metrics.SetGauge("Sys", float64(m.Sys))
	c.metrics.SetGauge("TotalAlloc", float64(m.TotalAlloc))

	// Добавляем дополнительные метрики
	c.pollCount++
	c.metrics.AddCounter("PollCount", 1) // увеличиваем на 1 при каждом сборе

	// RandomValue - произвольное значение
	randomValue := rand.Float64() * 100 // от 0 до 100
	c.metrics.SetGauge("RandomValue", randomValue)
}
