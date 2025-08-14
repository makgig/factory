package agent

import (
	"log"
	"time"
)

// Agent собирает и отправляет метрики
type Agent struct {
	serverURL string
	metrics   *MetricsStorage
	collector *Collector
	sender    *Sender
}

// New создает новый агент
func New(serverURL string) *Agent {
	metrics := NewMetricsStorage()
	collector := NewCollector(metrics)
	sender := NewSender(serverURL)

	return &Agent{
		serverURL: serverURL,
		metrics:   metrics,
		collector: collector,
		sender:    sender,
	}
}

// Run запускает агент в простом цикле
func (a *Agent) Run(pollInterval, reportInterval time.Duration) error {
	log.Printf("Агент запущен. Сбор метрик: %v, отправка: %v", pollInterval, reportInterval)

	var lastReport time.Time

	for {
		// Собираем метрики
		a.collector.CollectMetrics()
		log.Println("Runtime метрики собраны")

		// Проверяем нужно ли отправлять
		if time.Since(lastReport) >= reportInterval {
			metrics := a.metrics.GetAll()

			// Показываем текущее значение PollCount
			if pollCount, exists := metrics.Counters["PollCount"]; exists {
				log.Printf("PollCount = %d", pollCount)
			}

			log.Printf("Нужно отправить %d gauge и %d counter метрик",
				len(metrics.Gauges), len(metrics.Counters))

			// Отправляем метрики на сервер
			// if err := a.sender.SendMetrics(metrics); err != nil {
			// 	log.Printf("Ошибка отправки метрик: %v", err)
			// }

			a.sender.SendAll(metrics)

			lastReport = time.Now()
		}

		// Ждем до следующего сбора
		time.Sleep(pollInterval)
	}
}
