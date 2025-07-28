package agent

import (
	"bytes"
	"encoding/json"
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

	// Формируем URL для нового эндпоинта
	url := fmt.Sprintf("%s/update", s.serverURL)

	// Создаем запрос с JSON телом
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("создание запроса: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")

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
