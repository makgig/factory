package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/config"
	"github.com/makgig/factory/internal/handler"
	"github.com/makgig/factory/internal/repository"
	"github.com/makgig/factory/internal/service"
)

func main() {
	cfg, err := config.LoadServerConfig()

	if err != nil {
		log.Fatal("Ошибка конфигурации:", err)
	}

	// 1. Создаем хранилище
	storage := repository.New()

	// 2. Создаем сервис с хранилищем
	metricsService := service.New(storage)

	// 3. Создаем handler с сервисом
	h := handler.New(metricsService)

	// 4. Создаем роутер
	router := gin.Default()

	// 5. Настраиваем маршруты
	h.SetupRoutes(router)

	// 6. Запускаем сервер
	log.Printf("Запускаем сервер на %s", cfg.Address)
	if err := router.Run(cfg.Address); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}
