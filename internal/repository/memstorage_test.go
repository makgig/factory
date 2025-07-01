package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create new storage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := New()

			require.NotNil(t, storage, "New() should not return nil")
			require.NotNil(t, storage.gauges, "gauges map should not be nil")
			require.NotNil(t, storage.counters, "counters map should not be nil")
			assert.Empty(t, storage.gauges, "gauges map should be empty")
			assert.Empty(t, storage.counters, "counters map should be empty")
		})
	}
}

func TestMemStorage_UpdateGauge(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedGauges map[string]float64
	}{
		{
			name: "add new gauge metric",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			args: args{
				name:  "temperature",
				value: 23.5,
			},
			expectedGauges: map[string]float64{"temperature": 23.5},
		},
		{
			name: "update existing gauge metric",
			fields: fields{
				gauges:   map[string]float64{"temperature": 20.0},
				counters: map[string]int64{},
			},
			args: args{
				name:  "temperature",
				value: 25.0,
			},
			expectedGauges: map[string]float64{"temperature": 25.0},
		},
		{
			name: "add gauge to existing metrics",
			fields: fields{
				gauges:   map[string]float64{"cpu_usage": 75.5},
				counters: map[string]int64{},
			},
			args: args{
				name:  "memory_usage",
				value: 82.3,
			},
			expectedGauges: map[string]float64{
				"cpu_usage":    75.5,
				"memory_usage": 82.3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}

			m.UpdateGauge(tt.args.name, tt.args.value)

			assert.Equal(t, tt.expectedGauges, m.gauges, "gauges should match expected values")
		})
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		name  string
		value int64
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		expectedCounters map[string]int64
	}{
		{
			name: "add new counter metric",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			args: args{
				name:  "requests",
				value: 5,
			},
			expectedCounters: map[string]int64{"requests": 5},
		},
		{
			name: "add to existing counter metric",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 10},
			},
			args: args{
				name:  "requests",
				value: 3,
			},
			expectedCounters: map[string]int64{"requests": 13},
		},
		{
			name: "add different counter metric",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 5},
			},
			args: args{
				name:  "errors",
				value: 2,
			},
			expectedCounters: map[string]int64{"requests": 5, "errors": 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}

			m.UpdateCounter(tt.args.name, tt.args.value)

			assert.Equal(t, tt.expectedCounters, m.counters, "Counters should match expected values")
		})
	}
}
