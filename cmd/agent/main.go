package main

import (
	"flag"
	"log"
	"time"

	"github.com/makgig/factory/internal/agent"
)

func main() {
	// Парсим флаги
	address := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	reportInterval := flag.Int("r", 10, "частота отправки метрик на сервер (секунды)")
	pollInterval := flag.Int("p", 2, "частота опроса метрик из runtime (секунды)")
	flag.Parse()

	log.Println("Запускаем агент сбора метрик...")

	// Настройки агента
	pollIntervalDuration := time.Duration(*pollInterval) * time.Second
	reportIntervalDuration := time.Duration(*reportInterval) * time.Second
	serverURL := "http://" + *address

	log.Printf("Настройки агента:")
	log.Printf("- Сервер: %s", serverURL)
	log.Printf("- Интервал сбора: %v", pollIntervalDuration)
	log.Printf("- Интервал отправки: %v", reportIntervalDuration)

	// Создаем агент
	a := agent.New(serverURL)

	// Запускаем агент
	if err := a.Run(pollIntervalDuration, reportIntervalDuration); err != nil {
		log.Fatal("Ошибка работы агента:", err)
	}
}
