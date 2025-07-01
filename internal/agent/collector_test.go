package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCollector(t *testing.T) {
	type args struct {
		metrics *MetricsStorage
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "create collector with valid storage",
			args: args{
				metrics: NewMetricsStorage(),
			},
		},
		{
			name: "create collector with nil storage",
			args: args{
				metrics: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewCollector(tt.args.metrics)

			require.NotNil(t, collector, "NewCollector() should not return nil")
			assert.Equal(t, tt.args.metrics, collector.metrics, "metrics storage should match")
			assert.Equal(t, int64(0), collector.pollCount, "initial pollCount should be 0")
		})
	}
}

func TestCollector_CollectMetrics(t *testing.T) {
	type fields struct {
		metrics   *MetricsStorage
		pollCount int64
	}
	tests := []struct {
		name                     string
		fields                   fields
		expectedPollCount        int64
		expectedMetricsPollCount int64 // отдельно для метрик
	}{
		{
			name: "collect metrics with empty storage",
			fields: fields{
				metrics:   NewMetricsStorage(),
				pollCount: 0,
			},
			expectedPollCount:        1,
			expectedMetricsPollCount: 1, // первый вызов AddCounter
		},
		{
			name: "collect metrics multiple times",
			fields: fields{
				metrics:   NewMetricsStorage(),
				pollCount: 5,
			},
			expectedPollCount:        6, // 5 + 1
			expectedMetricsPollCount: 1, // только один вызов AddCounter в этом тесте
		},
		{
			name: "collect metrics with existing data",
			fields: fields{
				metrics: func() *MetricsStorage {
					storage := NewMetricsStorage()
					storage.SetGauge("existing_gauge", 123.45)
					storage.AddCounter("existing_counter", 10)
					// Предварительно добавляем PollCount
					storage.AddCounter("PollCount", 2) // симулируем 2 предыдущих вызова
					return storage
				}(),
				pollCount: 2,
			},
			expectedPollCount:        3, // 2 + 1
			expectedMetricsPollCount: 3, // 2 (предыдущие) + 1 (текущий)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Collector{
				metrics:   tt.fields.metrics,
				pollCount: tt.fields.pollCount,
			}

			// Выполняем сбор метрик
			c.CollectMetrics()

			// Проверяем что pollCount увеличился
			assert.Equal(t, tt.expectedPollCount, c.pollCount, "pollCount should be incremented")

			// Проверяем что PollCount добавлен в counters
			pollCountValue, exists := c.metrics.GetCounter("PollCount")
			assert.True(t, exists, "PollCount should exist in counters")
			assert.Equal(t, tt.expectedMetricsPollCount, pollCountValue, "PollCount value should match")

			// Проверяем что RandomValue добавлен в gauges
			randomValue, exists := c.metrics.GetGauge("RandomValue")
			assert.True(t, exists, "RandomValue should exist in gauges")
			assert.GreaterOrEqual(t, randomValue, 0.0, "RandomValue should be >= 0")
			assert.Less(t, randomValue, 100.0, "RandomValue should be < 100")

			// Проверяем наличие ключевых runtime метрик
			for _, metricName := range []string{"Alloc", "HeapAlloc", "Sys", "NumGC"} {
				_, exists := c.metrics.GetGauge(metricName)
				assert.True(t, exists, "runtime metric %s should exist", metricName)
			}
		})
	}
}
