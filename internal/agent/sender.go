package agent

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	// Формат: POST /update/gauge/<ИМЯ>/<ЗНАЧЕНИЕ>
	url := fmt.Sprintf("%s/update/gauge/%s/%s",
		s.serverURL, name, strconv.FormatFloat(value, 'f', -1, 64))

	return s.sendRequest(url)
}

// sendCounter отправляет одну counter метрику
func (s *Sender) sendCounter(name string, value int64) error {
	// Формат: POST /update/counter/<ИМЯ>/<ЗНАЧЕНИЕ>
	url := fmt.Sprintf("%s/update/counter/%s/%d",
		s.serverURL, name, value)

	return s.sendRequest(url)
}

// sendRequest выполняет HTTP POST запрос
func (s *Sender) sendRequest(url string) error {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("создание запроса: %w", err)
	}

	// Устанавливаем заголовок как в задании
	req.Header.Set("Content-Type", "text/plain")

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
