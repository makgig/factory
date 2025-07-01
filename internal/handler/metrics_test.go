package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/makgig/factory/internal/repository"
	"github.com/makgig/factory/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	type args struct {
		metricsService *service.MetricsService
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "create handler with valid service",
			args: args{
				metricsService: service.New(repository.New()),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := New(tt.args.metricsService)

			require.NotNil(t, handler, "New() should not return nil")
			assert.Equal(t, tt.args.metricsService, handler.metricsService, "handler should store the provided service")
		})
	}
}

func TestHandler_SetupRoutes(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		mux *http.ServeMux
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "setup routes with valid service and mux",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				mux: http.NewServeMux(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				metricsService: tt.fields.metricsService,
			}

			h.SetupRoutes(tt.args.mux)

			// Проверяем что роут /update/ настроен
			req, err := http.NewRequest("POST", "/update/gauge/test/1.0", nil)
			require.NoError(t, err, "should create request without error")

			rr := httptest.NewRecorder()
			tt.args.mux.ServeHTTP(rr, req)

			// Если роут настроен правильно, мы не должны получить 404
			assert.NotEqual(t, http.StatusNotFound, rr.Code, "route /update/ should be configured")
		})
	}
}

func TestHandler_updateMetricHandler(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "valid gauge metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("POST", "/update/gauge/temperature/23.5", nil),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name: "valid counter metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("POST", "/update/counter/requests/100", nil),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name: "method not allowed",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("GET", "/update/gauge/temperature/23.5", nil),
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed\n",
		},
		{
			name: "invalid URL format",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("POST", "/update/gauge/temperature", nil),
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Invalid URL format\n",
		},
		{
			name: "empty metric name",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("POST", "/update/gauge//23.5", nil),
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "metric name is required\n",
		},
		{
			name: "invalid metric type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("POST", "/update/invalid/temperature/23.5", nil),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid metric type\n",
		},
		{
			name: "invalid gauge value",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
				r: httptest.NewRequest("POST", "/update/gauge/temperature/not_a_number", nil),
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid gauge value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				metricsService: tt.fields.metricsService,
			}

			// Создаем свежий ResponseRecorder для каждого теста
			rr := httptest.NewRecorder()

			// Вызываем handler
			h.updateMetricHandler(rr, tt.args.r)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, rr.Code, "status code should match")
			assert.Equal(t, tt.expectedBody, rr.Body.String(), "response body should match")
		})
	}
}
