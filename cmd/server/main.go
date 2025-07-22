package main

import (
	"log"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/config"
	"github.com/makgig/factory/internal/handler"
	"github.com/makgig/factory/internal/logger"
	"github.com/makgig/factory/internal/repository"
	"github.com/makgig/factory/internal/service"
)

func main() {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		log.Fatal("Ошибка конфигурации:", err)
	}

	if err := logger.Initialize(cfg); err != nil {
		log.Fatal("Ошибка инициализации логера:", err)
	}

	// 1. Создаем хранилище
	storage := repository.New()

	// 2. Создаем сервис с хранилищем
	metricsService := service.New(storage)

	// 3. Создаем handler с сервисом
	h := handler.New(metricsService)

	// 4. Создаем роутер
	cfg.ApplyGinMode()
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger.GinLogger())

	// 5. Настраиваем маршруты
	h.SetupRoutes(router)

	// 6. Запускаем сервер
	logger.Log.Info("Запускаем сервер", zap.String("address", cfg.Address))
	if err := router.Run(cfg.Address); err != nil {
		logger.Log.Fatal("Ошибка запуска сервера", zap.Error(err))
	}
}
