package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/repository"
	"github.com/makgig/factory/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// init отключает логи Gin для всех тестов
func init() {
	gin.SetMode(gin.TestMode)
}

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
		router *gin.Engine
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
				router: gin.New(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				metricsService: tt.fields.metricsService,
			}

			h.SetupRoutes(tt.args.router)

			// Проверяем что роут /update/ настроен
			req, err := http.NewRequest("POST", "/update/gauge/test/1.0", nil)
			require.NoError(t, err, "should create request without error")

			rr := httptest.NewRecorder()
			tt.args.router.ServeHTTP(rr, req)

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
		method string
		url    string
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
				method: "POST",
				url:    "/update/gauge/temperature/23.5",
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
				method: "POST",
				url:    "/update/counter/requests/100",
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
				method: "GET",
				url:    "/update/gauge/temperature/23.5",
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name: "invalid URL format",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method: "POST",
				url:    "/update/gauge/temperature",
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found",
		},
		{
			name: "empty metric name",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method: "POST",
				url:    "/update/gauge//23.5",
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "metric name is required",
		},
		{
			name: "invalid metric type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method: "POST",
				url:    "/update/invalid/temperature/23.5",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid metric type",
		},
		{
			name: "invalid gauge value",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method: "POST",
				url:    "/update/gauge/temperature/not_a_number",
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid gauge value",
		},
		{
			name: "negative gauge value",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method: "POST",
				url:    "/update/gauge/temperature/-10.5",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name: "zero counter value",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method: "POST",
				url:    "/update/counter/requests/0",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем Gin роутер для каждого теста
			router := gin.New()
			h := &Handler{
				metricsService: tt.fields.metricsService,
			}
			h.SetupRoutes(router)
			// Создаем HTTP запрос
			req, err := http.NewRequest(tt.args.method, tt.args.url, nil)
			require.NoError(t, err, "should create request without error")

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Выполняем запрос
			router.ServeHTTP(rr, req)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, rr.Code, "status code should match")
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, rr.Body.String(), "response body should match")
			}
		})
	}
}
