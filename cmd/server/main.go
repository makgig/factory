package main

import (
	"fmt"
	"log"
	"os"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/config"
	"github.com/makgig/factory/internal/handler"
	"github.com/makgig/factory/internal/logger"
	"github.com/makgig/factory/internal/middleware"
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

	logger.Log.Info("Server configuration",
		zap.String("address", cfg.Address),
		zap.Duration("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)

	// 1. Создаем хранилище
	storage := repository.New(cfg.FileStoragePath)

	// 2. Настраиваем параметры сохранения
	storage.SetStoreConfig(cfg.StoreInterval, cfg.IsSyncStore())

	if cfg.IsSyncStore() {
		logger.Log.Info("Включен режим синхронного сохранения метрик")
	} else {
		logger.Log.Info("Включен режим периодического сохранения метрик",
			zap.Duration("interval", cfg.StoreInterval))
	}

	// 3. Загружаем сохраненные данные при старте (если нужно)
	if cfg.Restore {
		// Проверяем существует ли файл
		if _, err := os.Stat(cfg.FileStoragePath); err != nil {
			if os.IsNotExist(err) {
				logger.Log.Info("Файл метрик не найден, начинаем с пустого хранилища", zap.String("file", cfg.FileStoragePath))
			} else {
				logger.Log.Error("Ошибка проверки файла метрик", zap.Error(err))
			}
		} else {
			// Файл существует - загружаем
			if err := storage.LoadFromFile(); err != nil {
				logger.Log.Error("Ошибка загрузки метрик из файла", zap.Error(err))
			} else {
				logger.Log.Info("Метрики успешно загружены из файла", zap.String("file", cfg.FileStoragePath))
			}
		}
	} else {
		logger.Log.Info("Загрузка метрик отключена (RESTORE=false)")
	}

	// 4. Создаем сервис с хранилищем
	metricsService := service.New(storage)

	// 5. Создаем handler с сервисом
	h := handler.New(metricsService)

	// 6. Создаем роутер
	cfg.ApplyGinMode()
	router := gin.New()

	// 🔥 ОТЛАДКА 1: Логируем ВСЕ входящие запросы
	router.Use(func(c *gin.Context) {
		fmt.Printf("🌐 INCOMING REQUEST: %s %s\n", c.Request.Method, c.Request.URL.Path)
		c.Next()
	})

	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.Gzip())

	// 7. Настраиваем маршруты
	h.SetupRoutes(router)

	// 🔥 ОТЛАДКА 2: Выводим все зарегистрированные роуты
	fmt.Println("\n🚀 REGISTERED ROUTES:")
	for _, route := range router.Routes() {
		fmt.Printf("   %s %s\n", route.Method, route.Path)
	}
	fmt.Println()

	// 8. Сохраняем метрики при нормальном завершении
	defer func() {
		logger.Log.Info("Сохраняем метрики при завершении...")
		if err := storage.SaveToFile(); err != nil {
			logger.Log.Error("Ошибка сохранения метрик при завершении", zap.Error(err))
		} else {
			logger.Log.Info("Метрики сохранены при завершении", zap.String("file", cfg.FileStoragePath))
		}
	}()

	// 9. Запускаем сервер
	logger.Log.Info("Запускаем сервер", zap.String("address", cfg.Address))
	if err := router.Run(cfg.Address); err != nil {
		logger.Log.Fatal("Ошибка запуска сервера", zap.Error(err))
	}
}
