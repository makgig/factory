package main

import (
	"log"
	"net/http"

	"github.com/makgig/factory/internal/handler"
	"github.com/makgig/factory/internal/repository"
	"github.com/makgig/factory/internal/service"
)

func main() {
	// 1. Создаем хранилище
	storage := repository.New()

	// 2. Создаем сервис с хранилищем
	metricsService := service.New(storage)

	// 3. Создаем handler с сервисом
	h := handler.New(metricsService)

	// 4. Создаем роутер
	mux := http.NewServeMux()

	// 5. Настраиваем маршруты
	h.SetupRoutes(mux)

	// 6. Запускаем сервер
	log.Println("Запускаем сервер на :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}
