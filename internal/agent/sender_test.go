package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/makgig/factory/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSender(t *testing.T) {
	type args struct {
		serverURL string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "create sender with valid URL",
			args: args{
				serverURL: "http://localhost:8080",
			},
		},
		{
			name: "create sender with different URL",
			args: args{
				serverURL: "https://example.com",
			},
		},
		{
			name: "create sender with empty URL",
			args: args{
				serverURL: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewSender(tt.args.serverURL)

			require.NotNil(t, sender, "NewSender() should not return nil")
			assert.Equal(t, tt.args.serverURL, sender.serverURL, "serverURL should match")
			require.NotNil(t, sender.httpClient, "httpClient should not be nil")
			assert.Equal(t, 5*time.Second, sender.httpClient.Timeout, "timeout should be 5 seconds")
		})
	}
}

func TestSender_SendMetrics(t *testing.T) {
	// Создаем тестовый сервер для JSON API с gzip поддержкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем метод и заголовки
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/update" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Проверяем gzip заголовки
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"), "должен быть Content-Encoding: gzip")
		assert.Equal(t, "gzip", r.Header.Get("Accept-Encoding"), "должен быть Accept-Encoding: gzip")

		// Читаем и распаковываем gzip данные
		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer gzReader.Close()

		body, err := io.ReadAll(gzReader)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metric models.Metrics
		if err := json.Unmarshal(body, &metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Проверяем что метрика валидна
		if metric.ID == "" || metric.MType == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Возвращаем успех
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	type fields struct {
		serverURL  string
		httpClient *http.Client
	}
	type args struct {
		metrics AllMetrics
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "send valid metrics with gzip",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				metrics: AllMetrics{
					Gauges:   map[string]float64{"temperature": 23.5},
					Counters: map[string]int64{"requests": 100},
				},
			},
			wantErr: false,
		},
		{
			name: "send empty metrics",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				metrics: AllMetrics{
					Gauges:   map[string]float64{},
					Counters: map[string]int64{},
				},
			},
			wantErr: false,
		},
		{
			name: "send to invalid server",
			fields: fields{
				serverURL:  "http://invalid-server:99999",
				httpClient: &http.Client{Timeout: 1 * time.Second},
			},
			args: args{
				metrics: AllMetrics{
					Gauges:   map[string]float64{"temperature": 23.5},
					Counters: map[string]int64{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sender{
				serverURL:  tt.fields.serverURL,
				httpClient: tt.fields.httpClient,
			}

			err := s.SendMetrics(tt.args.metrics)

			if tt.wantErr {
				assert.Error(t, err, "should return an error")
			} else {
				assert.NoError(t, err, "should not return an error")
			}
		})
	}
}

func TestSender_sendGauge(t *testing.T) {
	// Создаем тестовый сервер для JSON API с gzip поддержкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем URL и метод
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/update" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Проверяем gzip заголовки
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))

		// Читаем и распаковываем gzip JSON
		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer gzReader.Close()

		body, err := io.ReadAll(gzReader)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metric models.Metrics
		if err := json.Unmarshal(body, &metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Проверяем что это gauge метрика
		if metric.MType != "gauge" || metric.Value == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	type fields struct {
		serverURL  string
		httpClient *http.Client
	}
	type args struct {
		name  string
		value float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "send valid gauge metric with gzip",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				name:  "temperature",
				value: 23.5,
			},
			wantErr: false,
		},
		{
			name: "send gauge with zero value",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				name:  "pressure",
				value: 0.0,
			},
			wantErr: false,
		},
		{
			name: "send gauge with negative value",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				name:  "temperature",
				value: -10.5,
			},
			wantErr: false,
		},
		{
			name: "send to invalid server",
			fields: fields{
				serverURL:  "http://invalid-server:99999",
				httpClient: &http.Client{Timeout: 1 * time.Second},
			},
			args: args{
				name:  "temperature",
				value: 23.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sender{
				serverURL:  tt.fields.serverURL,
				httpClient: tt.fields.httpClient,
			}

			err := s.sendGauge(tt.args.name, tt.args.value)

			if tt.wantErr {
				assert.Error(t, err, "should return an error")
			} else {
				assert.NoError(t, err, "should not return an error")
			}
		})
	}
}

func TestSender_sendCounter(t *testing.T) {
	// Создаем тестовый сервер для JSON API с gzip поддержкой
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/update" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Проверяем gzip заголовки
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))

		// Читаем и распаковываем gzip JSON
		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer gzReader.Close()

		body, err := io.ReadAll(gzReader)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var metric models.Metrics
		if err := json.Unmarshal(body, &metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Проверяем что это counter метрика
		if metric.MType != "counter" || metric.Delta == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	type fields struct {
		serverURL  string
		httpClient *http.Client
	}
	type args struct {
		name  string
		value int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "send valid counter metric with gzip",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				name:  "requests",
				value: 100,
			},
			wantErr: false,
		},
		{
			name: "send counter with zero value",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				name:  "errors",
				value: 0,
			},
			wantErr: false,
		},
		{
			name: "send to invalid server",
			fields: fields{
				serverURL:  "http://invalid-server:99999",
				httpClient: &http.Client{Timeout: 1 * time.Second},
			},
			args: args{
				name:  "requests",
				value: 100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sender{
				serverURL:  tt.fields.serverURL,
				httpClient: tt.fields.httpClient,
			}

			err := s.sendCounter(tt.args.name, tt.args.value)

			if tt.wantErr {
				assert.Error(t, err, "should return an error")
			} else {
				assert.NoError(t, err, "should not return an error")
			}
		})
	}
}

func TestSender_sendJSONMetric(t *testing.T) {
	// Создаем тестовый сервер с разными ответами для gzip
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/update":
			if r.Method != "POST" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if r.Header.Get("Content-Type") != "application/json" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Проверяем gzip заголовки
			contentEncoding := r.Header.Get("Content-Encoding")
			acceptEncoding := r.Header.Get("Accept-Encoding")

			if contentEncoding != "gzip" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if acceptEncoding != "gzip" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Распаковываем gzip данные
			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			defer gzReader.Close()

			body, err := io.ReadAll(gzReader)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Проверяем что JSON валиден
			var metric models.Metrics
			if err := json.Unmarshal(body, &metric); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		case "/error":
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Bad Request"))
		case "/server-error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}
	}))
	defer server.Close()

	type fields struct {
		serverURL  string
		httpClient *http.Client
	}
	type args struct {
		metric models.Metrics
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "send valid gauge metric with gzip",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				metric: models.Metrics{
					ID:    "temperature",
					MType: "gauge",
					Value: func() *float64 { v := 23.5; return &v }(),
				},
			},
			wantErr: false,
		},
		{
			name: "send valid counter metric with gzip",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				metric: models.Metrics{
					ID:    "requests",
					MType: "counter",
					Delta: func() *int64 { v := int64(100); return &v }(),
				},
			},
			wantErr: false,
		},
		{
			name: "send to invalid server",
			fields: fields{
				serverURL:  "http://invalid-server:99999",
				httpClient: &http.Client{Timeout: 1 * time.Second},
			},
			args: args{
				metric: models.Metrics{
					ID:    "temperature",
					MType: "gauge",
					Value: func() *float64 { v := 23.5; return &v }(),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sender{
				serverURL:  tt.fields.serverURL,
				httpClient: tt.fields.httpClient,
			}

			err := s.sendJSONMetric(tt.args.metric)

			if tt.wantErr {
				assert.Error(t, err, "should return an error")
			} else {
				assert.NoError(t, err, "should not return an error")
			}
		})
	}
}

func TestSender_GzipCompression(t *testing.T) {
	// Специальный тест для проверки что данные действительно сжимаются
	var receivedHeaders http.Header
	var receivedBody []byte
	var isCompressed bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		// Проверяем сжатие
		if r.Header.Get("Content-Encoding") == "gzip" {
			isCompressed = true
			// Читаем сжатые данные без распаковки
			body, _ := io.ReadAll(r.Body)
			receivedBody = body
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	sender := NewSender(server.URL)

	// Отправляем тестовую метрику
	metric := models.Metrics{
		ID:    "test_metric",
		MType: "gauge",
		Value: func() *float64 { v := 123.456; return &v }(),
	}

	err := sender.sendJSONMetric(metric)
	require.NoError(t, err)

	// Проверяем что данные были сжаты
	assert.True(t, isCompressed, "данные должны быть сжаты")
	assert.Equal(t, "gzip", receivedHeaders.Get("Content-Encoding"))
	assert.Equal(t, "gzip", receivedHeaders.Get("Accept-Encoding"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))

	// Проверяем что можем распаковать полученные данные
	gzReader, err := gzip.NewReader(bytes.NewReader(receivedBody))
	require.NoError(t, err)
	defer gzReader.Close()

	uncompressedData, err := io.ReadAll(gzReader)
	require.NoError(t, err)

	var unpackedMetric models.Metrics
	err = json.Unmarshal(uncompressedData, &unpackedMetric)
	require.NoError(t, err)

	// Проверяем что метрика корректна
	assert.Equal(t, "test_metric", unpackedMetric.ID)
	assert.Equal(t, "gauge", unpackedMetric.MType)
	assert.Equal(t, 123.456, *unpackedMetric.Value)
}
