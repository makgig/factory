package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/makgig/factory/internal/config"
	"github.com/makgig/factory/internal/handler"
	"github.com/makgig/factory/internal/logger"
	"github.com/makgig/factory/internal/middleware"
	"github.com/makgig/factory/internal/repository"
	"github.com/makgig/factory/internal/service"
	"go.uber.org/zap"
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
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.Gzip())

	// 7. Настраиваем маршруты
	h.SetupRoutes(router)

	// 8. Создаем HTTP сервер
	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: router,
	}

	// 9. Запускаем сервер в горутине
	go func() {
		logger.Log.Info("Запускаем сервер", zap.String("address", cfg.Address))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("Ошибка запуска сервера", zap.Error(err))
		}
	}()

	// 10. Настраиваем graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Блокируемся до получения сигнала
	<-quit
	logger.Log.Info("Получен сигнал завершения, останавливаем сервер...")

	// 11. Сохраняем метрики перед завершением
	logger.Log.Info("Сохраняем метрики при завершении...")
	if err := storage.SaveToFile(); err != nil {
		logger.Log.Error("Ошибка сохранения метрик при завершении", zap.Error(err))
	} else {
		logger.Log.Info("Метрики сохранены при завершении", zap.String("file", cfg.FileStoragePath))
	}

	// 12. Даем серверу 5 секунд на graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Принудительное завершение сервера", zap.Error(err))
	}

	logger.Log.Info("Сервер корректно завершен")
}
