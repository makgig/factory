package main

import (
	"log"

	"github.com/makgig/factory/internal/agent"
	"github.com/makgig/factory/internal/config"
)

func main() {
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.Fatal("Ошибка конфигурации:", err)
	}

	log.Printf("Настройки агента:")
	log.Printf("- Сервер: %s", cfg.ServerURL())
	log.Printf("- Интервал сбора: %v", cfg.PollInterval)
	log.Printf("- Интервал отправки: %v", cfg.ReportInterval)

	// Создаем агент
	a := agent.New(cfg.ServerURL())

	// Запускаем агент
	if err := a.Run(cfg.PollInterval, cfg.ReportInterval); err != nil {
		log.Fatal("Ошибка работы агента:", err)
	}
}
