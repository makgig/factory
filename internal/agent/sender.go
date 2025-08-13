package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/makgig/factory/internal/models"

	"net/http"
	"time"
)

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

func (s *Sender) sendJSONMetric(metric models.Metrics) error {
	// Сериализуем в JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	// Сжимаем JSON данные
	var compressedData bytes.Buffer
	gzWriter := gzip.NewWriter(&compressedData)
	if _, err := gzWriter.Write(jsonData); err != nil {
		return fmt.Errorf("ошибка сжатия данных: %w", err)
	}
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("ошибка финализации сжатия: %w", err)
	}

	// Формируем URL для нового эндпоинта
	url := fmt.Sprintf("%s/update", s.serverURL)

	// Создаем запрос с JSON телом
	req, err := http.NewRequest("POST", url, bytes.NewReader(compressedData.Bytes()))
	if err != nil {
		return fmt.Errorf("создание запроса: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// Выполняем запрос
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("выполнение запроса: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("сервер вернул статус %d", resp.StatusCode)
	}

	return nil
}

// SendBatch отправляет []Metrics на POST /updates/ с gzip.
// Пустые батчи НЕ отправляем (вернёт nil).
func (s *Sender) SendBatch(items []models.Metrics) error {
	if len(items) == 0 {
		return nil
	}

	// JSON
	body, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	// gzip
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	// Формируем URL
	url := s.serverURL + "/updates/"

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound, http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return ErrBatchUnsupported
	default:
		return fmt.Errorf("batch response status: %d", resp.StatusCode)
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
