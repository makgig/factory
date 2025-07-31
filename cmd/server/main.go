package main

import (
	"log"

	"github.com/makgig/factory/internal/config"
	"github.com/makgig/factory/internal/server"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadServerConfig()
	if err != nil {
		log.Fatal("Ошибка конфигурации:", err)
	}

	// Создаем и инициализируем сервер
	srv := server.New(cfg)
	if err := srv.Initialize(); err != nil {
		log.Fatal("Ошибка инициализации сервера:", err)
	}

	// Запускаем сервер (блокирующий вызов до завершения)
	if err := srv.Run(); err != nil {
		log.Fatal("Ошибка работы сервера:", err)
	}
}
