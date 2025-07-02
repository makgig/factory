package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsStorage(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "create new metrics storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMetricsStorage()

			require.NotNil(t, storage, "NewMetricsStorage() should not return nil")
			require.NotNil(t, storage.gauges, "gauges map should not be nil")
			require.NotNil(t, storage.counters, "counters map should not be nil")
			assert.Empty(t, storage.gauges, "gauges map should be empty")
			assert.Empty(t, storage.counters, "counters map should be empty")
		})
	}
}

func TestMetricsStorage_SetGauge(t *testing.T) {
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
			name: "set new gauge metric",
			fields: fields{
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
			},
			args: args{
				name:  "temperature",
				value: 23.5,
			},
			expectedGauges: map[string]float64{"temperature": 23.5},
		},
		{
			name: "replace existing gauge metric",
			fields: fields{
				gauges:   map[string]float64{"temperature": 20.0},
				counters: make(map[string]int64),
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
				counters: make(map[string]int64),
			},
			args: args{
				name:  "memory_usage",
				value: 82.3,
			},
			expectedGauges: map[string]float64{"cpu_usage": 75.5, "memory_usage": 82.3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricsStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}

			m.SetGauge(tt.args.name, tt.args.value)

			assert.Equal(t, tt.expectedGauges, m.gauges, "gauges should match expected values")
		})
	}
}

func TestMetricsStorage_AddCounter(t *testing.T) {
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
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
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
				gauges:   make(map[string]float64),
				counters: map[string]int64{"requests": 10},
			},
			args: args{
				name:  "requests",
				value: 3,
			},
			expectedCounters: map[string]int64{"requests": 13}, // 10 + 3
		},
		{
			name: "add different counter metric",
			fields: fields{
				gauges:   make(map[string]float64),
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
			m := &MetricsStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}

			m.AddCounter(tt.args.name, tt.args.value)

			assert.Equal(t, tt.expectedCounters, m.counters, "counters should match expected values")
		})
	}
}

func TestMetricsStorage_GetGauge(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float64
		want1  bool
	}{
		{
			name: "get existing gauge metric",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: make(map[string]int64),
			},
			args: args{
				name: "temperature",
			},
			want:  23.5,
			want1: true,
		},
		{
			name: "get non-existing gauge metric",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: make(map[string]int64),
			},
			args: args{
				name: "pressure",
			},
			want:  0.0,
			want1: false,
		},
		{
			name: "get from empty storage",
			fields: fields{
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
			},
			args: args{
				name: "temperature",
			},
			want:  0.0,
			want1: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricsStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}

			got, got1 := m.GetGauge(tt.args.name)

			assert.Equal(t, tt.want, got, "gauge value should match")
			assert.Equal(t, tt.want1, got1, "gauge exists flag should match")
		})
	}
}

func TestMetricsStorage_GetCounter(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int64
		want1  bool
	}{
		{
			name: "get existing counter metric",
			fields: fields{
				gauges:   make(map[string]float64),
				counters: map[string]int64{"requests": 100},
			},
			args: args{
				name: "requests",
			},
			want:  100,
			want1: true,
		},
		{
			name: "get non-existing counter metric",
			fields: fields{
				gauges:   make(map[string]float64),
				counters: map[string]int64{"requests": 100},
			},
			args: args{
				name: "errors",
			},
			want:  0,
			want1: false,
		},
		{
			name: "get from empty storage",
			fields: fields{
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
			},
			args: args{
				name: "requests",
			},
			want:  0,
			want1: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricsStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}

			got, got1 := m.GetCounter(tt.args.name)

			assert.Equal(t, tt.want, got, "counter value should match")
			assert.Equal(t, tt.want1, got1, "counter exists flag should match")
		})
	}
}

func TestMetricsStorage_GetAll(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
		want   AllMetrics
	}{
		{
			name: "get all from populated storage",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5, "cpu_usage": 75.5},
				counters: map[string]int64{"requests": 100, "errors": 5},
			},
			want: AllMetrics{
				Gauges:   map[string]float64{"temperature": 23.5, "cpu_usage": 75.5},
				Counters: map[string]int64{"requests": 100, "errors": 5},
			},
		},
		{
			name: "get all from empty storage",
			fields: fields{
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
			},
			want: AllMetrics{
				Gauges:   map[string]float64{},
				Counters: map[string]int64{},
			},
		},
		{
			name: "get all with only gauges",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: make(map[string]int64),
			},
			want: AllMetrics{
				Gauges:   map[string]float64{"temperature": 23.5},
				Counters: map[string]int64{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MetricsStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}

			got := m.GetAll()

			assert.Equal(t, tt.want, got, "AllMetrics should match expected values")
		})
	}
}
