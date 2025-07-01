package main

import (
	"log"
	"time"

	"github.com/makgig/factory/internal/agent"
)

func main() {
	log.Println("Запускаем агент сбора метрик...")

	// Настройки агента
	pollInterval := 2 * time.Second    // собираем метрики каждые 2 секунды
	reportInterval := 10 * time.Second // отправляем каждые 10 секунд
	serverURL := "http://localhost:8080"

	// Создаем агент
	a := agent.New(serverURL)

	// Запускаем агент
	if err := a.Run(pollInterval, reportInterval); err != nil {
		log.Fatal("Ошибка работы агента:", err)
	}
}
