package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/handler"
	"github.com/makgig/factory/internal/repository"
	"github.com/makgig/factory/internal/service"
)

func main() {
	// Парсим флаги
	address := flag.String("a", "localhost:8080", "адрес эндпоинта HTTP-сервера")
	flag.Parse()

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
	log.Printf("Запускаем сервер на %s", *address)
	if err := router.Run(*address); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}
