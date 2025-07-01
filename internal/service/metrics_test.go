package service

import (
	"testing"

	"github.com/makgig/factory/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	type args struct {
		storage *repository.MemStorage
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "create service with valid storage",
			args: args{
				storage: repository.New(),
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
		storage *repository.MemStorage
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
				storage: repository.New(),
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
				storage: repository.New(),
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
				storage: repository.New(),
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
				storage: repository.New(),
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
				storage: repository.New(),
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
				storage: repository.New(),
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
				storage: repository.New(),
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
				storage: repository.New(),
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
