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
			require.NotNil(t, storage.GetAllGauges(), "gauges map should not be nil")
			require.NotNil(t, storage.GetAllCounters(), "counters map should not be nil")
			assert.Empty(t, storage.GetAllGauges(), "gauges map should be empty")
			assert.Empty(t, storage.GetAllCounters(), "counters map should be empty")
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

func TestMemStorage_GetGauge(t *testing.T) {
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
			name: "get existing gauge",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: map[string]int64{},
			},
			args: args{
				name: "temperature",
			},
			want:  23.5,
			want1: true,
		},
		{
			name: "get non-existing gauge",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: map[string]int64{},
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
				gauges:   map[string]float64{},
				counters: map[string]int64{},
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
			m := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			got, got1 := m.GetGauge(tt.args.name)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func TestMemStorage_GetCounter(t *testing.T) {
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
			name: "get existing counter",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100},
			},
			args: args{
				name: "requests",
			},
			want:  100,
			want1: true,
		},
		{
			name: "get non-existing counter",
			fields: fields{
				gauges:   map[string]float64{},
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
				gauges:   map[string]float64{},
				counters: map[string]int64{},
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
			m := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			got, got1 := m.GetCounter(tt.args.name)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func TestMemStorage_GetAllGauges(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]float64
	}{
		{
			name: "get all from populated storage",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5, "cpu_usage": 75.5},
				counters: map[string]int64{},
			},
			want: map[string]float64{"temperature": 23.5, "cpu_usage": 75.5},
		},
		{
			name: "get all from empty storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			want: map[string]float64{},
		},
		{
			name: "get all with single gauge",
			fields: fields{
				gauges:   map[string]float64{"temperature": 23.5},
				counters: map[string]int64{},
			},
			want: map[string]float64{"temperature": 23.5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			got := m.GetAllGauges()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMemStorage_GetAllCounters(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]int64
	}{
		{
			name: "get all from populated storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100, "errors": 5},
			},
			want: map[string]int64{"requests": 100, "errors": 5},
		},
		{
			name: "get all from empty storage",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
			want: map[string]int64{},
		},
		{
			name: "get all with single counter",
			fields: fields{
				gauges:   map[string]float64{},
				counters: map[string]int64{"requests": 100},
			},
			want: map[string]int64{"requests": 100},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			got := m.GetAllCounters()
			assert.Equal(t, tt.want, got)
		})
	}
}
