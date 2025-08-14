package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"

	"github.com/makgig/factory/internal/models"

	"net/http"
	"time"
)

const (
	netRetryDelayShort  = 1 * time.Second
	netRetryDelayMedium = 3 * time.Second
	netRetryDelayLong   = 5 * time.Second
)

var netRetrySchedule = []time.Duration{
	netRetryDelayShort,
	netRetryDelayMedium,
	netRetryDelayLong,
}

// Sender отправляет метрики на сервер по HTTP
type Sender struct {
	serverURL  string
	httpClient *http.Client
}

var ErrBatchUnsupported = errors.New("batch api unsupported by server")

// NewSender создает новый отправщик
func NewSender(serverURL string) *Sender {
	return &Sender{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // таймаут запросов
		},
	}
}

// SendMetrics отправляет все метрики на сервер
func (s *Sender) SendMetrics(metrics AllMetrics) error {
	// Отправляем все gauge метрики
	for name, value := range metrics.Gauges {
		if err := s.sendGauge(name, value); err != nil {
			log.Printf("Ошибка отправки gauge %s: %v", name, err)
			return err
		}
	}

	// Отправляем все counter метрики
	for name, value := range metrics.Counters {
		if err := s.sendCounter(name, value); err != nil {
			log.Printf("Ошибка отправки counter %s: %v", name, err)
			return err
		}
	}

	log.Printf("Успешно отправлено %d gauge и %d counter метрик",
		len(metrics.Gauges), len(metrics.Counters))
	return nil
}

// sendGauge отправляет одну gauge метрику
func (s *Sender) sendGauge(name string, value float64) error {
	metric := models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}

	return s.sendJSONMetric(metric)
}

// sendCounter отправляет одну counter метрику
func (s *Sender) sendCounter(name string, value int64) error {
	metric := models.Metrics{
		ID:    name,
		MType: "counter",
		Delta: &value,
	}

	return s.sendJSONMetric(metric)
}

func (s *Sender) sendJSONMetric(m models.Metrics) error {
	raw, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}
	payload := buf.Bytes()

	delays := netRetrySchedule

	url := fmt.Sprintf("%s/update", s.serverURL)

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("new request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.ContentLength = int64(len(payload))
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(payload)), nil }

		resp, err := s.httpClient.Do(req)
		if err != nil {
			if isNetRetriable(err) && attempt < len(delays) {
				time.Sleep(delays[attempt])
				continue
			}
			return fmt.Errorf("metric do: %w", err)
		}
		resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			if attempt < len(delays) {
				time.Sleep(delays[attempt])
				continue
			}
			return fmt.Errorf("metric http %d", resp.StatusCode)
		default:
			return fmt.Errorf("metric response status: %d", resp.StatusCode)
		}
	}
}

// SendBatch отправляет []Metrics на POST /updates/ с gzip.
// Пустые батчи НЕ отправляем (вернёт nil).
func (s *Sender) SendBatch(items []models.Metrics) error {
	if len(items) == 0 {
		return nil
	}

	raw, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}
	payload := buf.Bytes()

	delays := netRetrySchedule
	var last error

	url := fmt.Sprintf("%s/updates/", s.serverURL)

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("new request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.ContentLength = int64(len(payload))
		req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(payload)), nil }

		resp, err := s.httpClient.Do(req)
		if err != nil {
			if isNetRetriable(err) && attempt < len(delays) {
				last = fmt.Errorf("batch do: %w", err)
				time.Sleep(delays[attempt])
				continue
			}
			return fmt.Errorf("batch do: %w", err)
		}
		func() {
			defer resp.Body.Close()
			switch resp.StatusCode {
			case http.StatusOK:
				last = nil
			case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusNotImplemented:
				last = ErrBatchUnsupported
			case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
				if attempt < len(delays) {
					last = fmt.Errorf("batch http %d", resp.StatusCode)
				} else {
					last = fmt.Errorf("batch http %d", resp.StatusCode)
				}
			default:
				last = fmt.Errorf("batch response status: %d", resp.StatusCode)
			}
		}()
		if last == nil || errors.Is(last, ErrBatchUnsupported) || attempt >= len(delays) {
			return last
		}
		time.Sleep(delays[attempt])
	}
}

func (s *Sender) SendAll(all AllMetrics) {
	// 1) собрать []models.Metrics
	var batch []models.Metrics
	for name, v := range all.Gauges {
		val := v
		batch = append(batch, models.Metrics{ID: name, MType: "gauge", Value: &val})
	}
	for name, d := range all.Counters {
		delta := d
		batch = append(batch, models.Metrics{ID: name, MType: "counter", Delta: &delta})
	}

	// 2) попытка батчем
	log.Println("Попытка отправки метрик батчем...")
	if err := s.SendBatch(batch); err != nil {
		if errors.Is(err, ErrBatchUnsupported) {
			// Если батчевый API не поддерживается, переключаемся на старый метод
			log.Println("Батчевый API не поддерживается, переключение на отправку по одной метрике.")
			// 3) фолбэк — по старому API /update
			for _, m := range batch {
				if err := s.sendJSONMetric(m); err != nil {
					log.Printf("Не удалось отправить метрику %s: %v", m.ID, err)
				}
			}
			return
		}
		// Если произошла другая ошибка при батчевой отправке
		log.Printf("Батчевая отправка не удалась из-за ошибки: %v. Попытка отправки по одной метрике.", err)
		for _, m := range batch {
			if err := s.sendJSONMetric(m); err != nil {
				log.Printf("Не удалось отправить метрику %s: %v", m.ID, err)
			}
		}
		return
	}

	log.Println("Метрики успешно отправлены батчем.")
}

func isNetRetriable(err error) bool {
	if err == nil {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, context.DeadlineExceeded)
}
