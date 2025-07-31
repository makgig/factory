package service

import (
	"testing"

	"github.com/makgig/factory/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	type args struct {
		storage repository.Storage
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "create service with valid storage",
			args: args{
				storage: repository.New(""),
			},
		},
		{
			name: "create service with nil storage",
			args: args{
				storage: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := New(tt.args.storage)

			require.NotNil(t, service, "New() should not return nil")
			assert.Equal(t, tt.args.storage, service.storage, "service should store the provided storage")
		})
	}
}

func TestMetricsService_UpdateMetric(t *testing.T) {
	type fields struct {
		storage repository.Storage
	}
	type args struct {
		metricType string
		name       string
		value      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "valid gauge metric",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "gauge",
				name:       "temperature",
				value:      "23.5",
			},
			wantErr: false,
		},
		{
			name: "valid counter metric",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "counter",
				name:       "requests",
				value:      "100",
			},
			wantErr: false,
		},
		{
			name: "empty metric name",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "gauge",
				name:       "",
				value:      "23.5",
			},
			wantErr: true,
		},
		{
			name: "invalid metric type",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "invalid",
				name:       "test",
				value:      "123",
			},
			wantErr: true,
		},
		{
			name: "invalid gauge value",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "gauge",
				name:       "temperature",
				value:      "not_a_number",
			},
			wantErr: true,
		},
		{
			name: "invalid counter value",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "counter",
				name:       "requests",
				value:      "not_a_number",
			},
			wantErr: true,
		},
		{
			name: "negative gauge value",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "gauge",
				name:       "temperature",
				value:      "-10.5",
			},
			wantErr: false,
		},
		{
			name: "zero counter value",
			fields: fields{
				storage: repository.New(""),
			},
			args: args{
				metricType: "counter",
				name:       "requests",
				value:      "0",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{
				storage: tt.fields.storage,
			}

			err := s.UpdateMetric(tt.args.metricType, tt.args.name, tt.args.value)

			if tt.wantErr {
				assert.Error(t, err, "should return an error")
			} else {
				assert.NoError(t, err, "should not return an error")
			}
		})
	}
}

func TestMetricsService_GetMetric(t *testing.T) {
	type fields struct {
		storage repository.Storage
	}
	type args struct {
		metricType string
		name       string
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		setupData func(repository.Storage)
		want      string
		wantErr   bool
	}{
		{
			name: "get existing gauge metric",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {
				storage.UpdateGauge("temperature", 23.5)
			},
			args: args{
				metricType: "gauge",
				name:       "temperature",
			},
			want:    "23.5",
			wantErr: false,
		},
		{
			name: "get existing counter metric",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {
				storage.UpdateCounter("requests", 100)
			},
			args: args{
				metricType: "counter",
				name:       "requests",
			},
			want:    "100",
			wantErr: false,
		},
		{
			name: "get non-existing gauge metric",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {},
			args: args{
				metricType: "gauge",
				name:       "nonexistent",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "get non-existing counter metric",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {},
			args: args{
				metricType: "counter",
				name:       "nonexistent",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid metric type",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {},
			args: args{
				metricType: "invalid",
				name:       "test",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "empty metric name",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {},
			args: args{
				metricType: "gauge",
				name:       "",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем данные
			if tt.setupData != nil {
				tt.setupData(tt.fields.storage)
			}

			s := &MetricsService{
				storage: tt.fields.storage,
			}
			got, err := s.GetMetric(tt.args.metricType, tt.args.name)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMetricsService_GetAllMetrics(t *testing.T) {
	type fields struct {
		storage repository.Storage
	}
	tests := []struct {
		name      string
		fields    fields
		setupData func(repository.Storage)
		want      MetricsData
	}{
		{
			name: "get all metrics from populated storage",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {
				storage.UpdateGauge("temperature", 23.5)
				storage.UpdateGauge("cpu_usage", 75.5)
				storage.UpdateCounter("requests", 100)
				storage.UpdateCounter("errors", 5)
			},
			want: MetricsData{
				Gauges:   map[string]float64{"temperature": 23.5, "cpu_usage": 75.5},
				Counters: map[string]int64{"requests": 100, "errors": 5},
			},
		},
		{
			name: "get all metrics from empty storage",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {},
			want: MetricsData{
				Gauges:   map[string]float64{},
				Counters: map[string]int64{},
			},
		},
		{
			name: "get all metrics with only gauges",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {
				storage.UpdateGauge("temperature", 23.5)
				storage.UpdateGauge("pressure", 1013.25)
			},
			want: MetricsData{
				Gauges:   map[string]float64{"temperature": 23.5, "pressure": 1013.25},
				Counters: map[string]int64{},
			},
		},
		{
			name: "get all metrics with only counters",
			fields: fields{
				storage: repository.New(""),
			},
			setupData: func(storage repository.Storage) {
				storage.UpdateCounter("requests", 100)
				storage.UpdateCounter("errors", 5)
			},
			want: MetricsData{
				Gauges:   map[string]float64{},
				Counters: map[string]int64{"requests": 100, "errors": 5},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем данные
			if tt.setupData != nil {
				tt.setupData(tt.fields.storage)
			}

			s := &MetricsService{
				storage: tt.fields.storage,
			}
			got := s.GetAllMetrics()
			assert.Equal(t, tt.want, got)
		})
	}
}
