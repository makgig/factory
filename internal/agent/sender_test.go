package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем метод и заголовки
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Возвращаем успех
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
			name: "send valid metrics",
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
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем URL и метод
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
			name: "send valid gauge metric",
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
	// Создаем тестовый сервер
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
			name: "send valid counter metric",
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

func TestSender_sendRequest(t *testing.T) {
	// Создаем тестовый сервер с разными ответами
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
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
		url string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successful request",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				url: server.URL + "/success",
			},
			wantErr: false,
		},
		{
			name: "bad request",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				url: server.URL + "/error",
			},
			wantErr: true,
		},
		{
			name: "server error",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 5 * time.Second},
			},
			args: args{
				url: server.URL + "/server-error",
			},
			wantErr: true,
		},
		{
			name: "invalid URL",
			fields: fields{
				serverURL:  server.URL,
				httpClient: &http.Client{Timeout: 1 * time.Second},
			},
			args: args{
				url: "invalid-url",
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

			err := s.sendRequest(tt.args.url)

			if tt.wantErr {
				assert.Error(t, err, "should return an error")
			} else {
				assert.NoError(t, err, "should not return an error")
			}
		})
	}
}
