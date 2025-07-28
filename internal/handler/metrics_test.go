package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

			// Проверяем что роуты настроены
			routes := tt.args.router.Routes()
			require.True(t, len(routes) > 0, "routes should be configured")

			// Проверим наличие основных роутов
			routePaths := make(map[string]bool)
			for _, route := range routes {
				routePaths[route.Method+" "+route.Path] = true
			}

			assert.True(t, routePaths["GET /"], "GET / route should be configured")
			assert.True(t, routePaths["GET /value/:type/:name"], "GET /value route should be configured")
			assert.True(t, routePaths["POST /update/:type/:name/:value"], "POST /update route should be configured")
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

func TestHandler_getMetricHandler(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		c *gin.Context
		w *httptest.ResponseRecorder
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "get existing gauge metric",
			fields: fields{
				metricsService: func() *service.MetricsService {
					s := service.New(repository.New())
					s.UpdateMetric("gauge", "temperature", "23.5")
					return s
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "23.5",
		},
		{
			name: "get non-existing metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				w: httptest.NewRecorder(),
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Metric not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем контекст с сохраненным ResponseRecorder
			c, _ := gin.CreateTestContext(tt.args.w)
			c.Params = gin.Params{
				{Key: "type", Value: "gauge"},
				{Key: "name", Value: "temperature"},
			}
			if tt.name == "get non-existing metric" {
				c.Params = gin.Params{
					{Key: "type", Value: "gauge"},
					{Key: "name", Value: "nonexistent"},
				}
			}

			h := &Handler{
				metricsService: tt.fields.metricsService,
			}
			h.getMetricHandler(c)

			assert.Equal(t, tt.expectedStatus, tt.args.w.Code)
			assert.Equal(t, tt.expectedBody, tt.args.w.Body.String())
		})
	}
}

func TestHandler_getAllMetricsHandler(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		c *gin.Context
		w *httptest.ResponseRecorder
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		expectedStatus      int
		expectedContentType string
		shouldContain       []string
	}{
		{
			name: "get all metrics with data",
			fields: fields{
				metricsService: func() *service.MetricsService {
					s := service.New(repository.New())
					s.UpdateMetric("gauge", "temperature", "23.5")
					s.UpdateMetric("counter", "requests", "100")
					return s
				}(),
			},
			args: args{
				w: httptest.NewRecorder(),
			},
			expectedStatus:      http.StatusOK,
			expectedContentType: "text/html; charset=utf-8",
			shouldContain:       []string{"temperature: 23.5", "requests: 100"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(tt.args.w)

			h := &Handler{
				metricsService: tt.fields.metricsService,
			}
			h.getAllMetricsHandler(c)

			assert.Equal(t, tt.expectedStatus, tt.args.w.Code)
			assert.Equal(t, tt.expectedContentType, tt.args.w.Header().Get("Content-Type"))

			body := tt.args.w.Body.String()
			for _, content := range tt.shouldContain {
				assert.Contains(t, body, content)
			}
		})
	}
}

func TestHandler_updateMetricJSONHandler(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		method      string
		url         string
		contentType string
		body        string
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
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"gauge","value":23.5}`,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name: "valid counter metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"requests","type":"counter","delta":100}`,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name: "missing content type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "",
				body:        `{"id":"temperature","type":"gauge","value":23.5}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Content-Type must be application/json"}`,
		},
		{
			name: "wrong content type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "text/plain",
				body:        `{"id":"temperature","type":"gauge","value":23.5}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Content-Type must be application/json"}`,
		},
		{
			name: "invalid JSON format",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"gauge","value":}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid JSON format"}`,
		},
		{
			name: "missing metric id",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"type":"gauge","value":23.5}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"metric id is required"}`,
		},
		{
			name: "missing metric type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"temperature","value":23.5}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"metric type is required"}`,
		},
		{
			name: "invalid metric type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"invalid","value":23.5}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"invalid metric type"}`,
		},
		{
			name: "gauge without value",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"gauge"}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"value is required for gauge metric"}`,
		},
		{
			name: "counter without delta",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"requests","type":"counter"}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"delta is required for counter metric"}`,
		},
		{
			name: "negative gauge value",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"gauge","value":-10.5}`,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name: "zero counter delta",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			args: args{
				method:      "POST",
				url:         "/update",
				contentType: "application/json",
				body:        `{"id":"requests","type":"counter","delta":0}`,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
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

			// Создаем HTTP запрос с телом
			req, err := http.NewRequest(tt.args.method, tt.args.url, strings.NewReader(tt.args.body))
			require.NoError(t, err, "should create request without error")

			// Устанавливаем Content-Type если указан
			if tt.args.contentType != "" {
				req.Header.Set("Content-Type", tt.args.contentType)
			}

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Выполняем запрос
			router.ServeHTTP(rr, req)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, rr.Code, "status code should match")
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, rr.Body.String(), "response body should match")
			}
		})
	}
}

func TestHandler_getMetricJSONHandler(t *testing.T) {
	type fields struct {
		metricsService *service.MetricsService
	}
	type args struct {
		method      string
		url         string
		contentType string
		body        string
	}
	tests := []struct {
		name           string
		fields         fields
		setupData      func(*service.MetricsService)
		args           args
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "get existing gauge metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {
				s.UpdateMetric("gauge", "temperature", "23.5")
			},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"gauge"}`,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"temperature","type":"gauge","value":23.5}`,
		},
		{
			name: "get existing counter metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {
				s.UpdateMetric("counter", "requests", "100")
			},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"requests","type":"counter"}`,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"requests","type":"counter","delta":100}`,
		},
		{
			name: "missing content type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "",
				body:        `{"id":"temperature","type":"gauge"}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Content-Type must be application/json"}`,
		},
		{
			name: "wrong content type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "text/plain",
				body:        `{"id":"temperature","type":"gauge"}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Content-Type must be application/json"}`,
		},
		{
			name: "invalid JSON format",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"temperature","type":}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid JSON format"}`,
		},
		{
			name: "missing metric id",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"type":"gauge"}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"metric id is required"}`,
		},
		{
			name: "missing metric type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"temperature"}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"metric type is required"}`,
		},
		{
			name: "invalid metric type",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"invalid"}`,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"error":"Invalid metric type"}`,
		},
		{
			name: "non-existing gauge metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"nonexistent","type":"gauge"}`,
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Metric not found"}`,
		},
		{
			name: "non-existing counter metric",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"nonexistent","type":"counter"}`,
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"error":"Metric not found"}`,
		},
		{
			name: "get gauge with negative value",
			fields: fields{
				metricsService: service.New(repository.New()),
			},
			setupData: func(s *service.MetricsService) {
				s.UpdateMetric("gauge", "temperature", "-10.5")
			},
			args: args{
				method:      "POST",
				url:         "/value",
				contentType: "application/json",
				body:        `{"id":"temperature","type":"gauge"}`,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"temperature","type":"gauge","value":-10.5}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготавливаем данные если нужно
			if tt.setupData != nil {
				tt.setupData(tt.fields.metricsService)
			}

			// Создаем Gin роутер для каждого теста
			router := gin.New()
			h := &Handler{
				metricsService: tt.fields.metricsService,
			}
			h.SetupRoutes(router)

			// Создаем HTTP запрос с телом
			req, err := http.NewRequest(tt.args.method, tt.args.url, strings.NewReader(tt.args.body))
			require.NoError(t, err, "should create request without error")

			// Устанавливаем Content-Type если указан
			if tt.args.contentType != "" {
				req.Header.Set("Content-Type", tt.args.contentType)
			}

			// Создаем ResponseRecorder
			rr := httptest.NewRecorder()

			// Выполняем запрос
			router.ServeHTTP(rr, req)

			// Проверяем результат
			assert.Equal(t, tt.expectedStatus, rr.Code, "status code should match")
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, rr.Body.String(), "response body should match")
			}
		})
	}
}
