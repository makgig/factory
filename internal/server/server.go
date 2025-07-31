package server

import (
	"context"
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

// Server представляет HTTP сервер метрик
type Server struct {
	cfg            *config.ServerConfig
	storage        repository.Storage
	metricsService *service.MetricsService
	httpServer     *http.Server
	router         *gin.Engine
}

// New создает новый экземпляр сервера
func New(cfg *config.ServerConfig) *Server {
	return &Server{
		cfg: cfg,
	}
}

// Initialize инициализирует все компоненты сервера
func (s *Server) Initialize() error {
	// Инициализируем логер
	if err := s.initializeLogger(); err != nil {
		return err
	}

	// Инициализируем хранилище
	if err := s.initializeStorage(); err != nil {
		return err
	}

	// Инициализируем сервисы
	s.initializeServices()

	// Настраиваем HTTP роутер
	s.setupRouter()

	// Создаем HTTP сервер
	s.createHTTPServer()

	logger.Log.Info("Сервер успешно инициализирован")
	return nil
}

// Run запускает сервер и блокируется до получения сигнала завершения
func (s *Server) Run() error {
	// Запускаем сервер в горутине
	errChan := make(chan error, 1)
	go func() {
		logger.Log.Info("Запускаем сервер", zap.String("address", s.cfg.Address))
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Настраиваем graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Ждем либо ошибку запуска, либо сигнал завершения
	select {
	case err := <-errChan:
		return err
	case <-quit:
		// Graceful shutdown с таймаутом 5 секунд
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.Stop(ctx)
	}
}

// Stop останавливает сервер с graceful shutdown
func (s *Server) Stop(ctx context.Context) error {
	logger.Log.Info("Получен сигнал завершения, останавливаем сервер...")

	// Останавливаем периодическое сохранение
	s.stopPeriodicSaving()

	// Сохраняем финальные метрики
	s.saveFinalMetrics()

	// Останавливаем HTTP сервер
	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Log.Error("Ошибка graceful shutdown", zap.Error(err))
		return err
	}

	logger.Log.Info("Сервер корректно завершен")
	return nil
}

// initializeLogger инициализирует систему логирования
func (s *Server) initializeLogger() error {
	if err := logger.Initialize(s.cfg); err != nil {
		return err
	}

	logger.Log.Info("Server configuration",
		zap.String("address", s.cfg.Address),
		zap.Duration("store_interval", s.cfg.StoreInterval),
		zap.String("file_storage_path", s.cfg.FileStoragePath),
		zap.Bool("restore", s.cfg.Restore),
	)

	return nil
}

// initializeStorage настраивает хранилище метрик
func (s *Server) initializeStorage() error {
	// Создаем хранилище
	s.storage = repository.New(s.cfg.FileStoragePath)

	// Настраиваем параметры сохранения
	s.storage.SetStoreConfig(s.cfg.StoreInterval, s.cfg.IsSyncStore())

	// Логируем режим сохранения
	s.logSaveMode()

	// Запускаем периодическое сохранение (если нужно)
	s.startPeriodicSaving()

	// Загружаем данные (если нужно)
	return s.loadStoredData()
}

// logSaveMode логирует режим сохранения
func (s *Server) logSaveMode() {
	if s.cfg.IsSyncStore() {
		logger.Log.Info("Включен режим синхронного сохранения метрик")
	} else {
		logger.Log.Info("Включен режим периодического сохранения метрик",
			zap.Duration("interval", s.cfg.StoreInterval))
	}
}

// startPeriodicSaving запускает периодическое сохранение
func (s *Server) startPeriodicSaving() {
	if !s.cfg.IsSyncStore() && s.cfg.StoreInterval > 0 && s.cfg.FileStoragePath != "" {
		s.storage.StartSavingLoop()
		logger.Log.Info("Запущено периодическое сохранение метрик",
			zap.Duration("interval", s.cfg.StoreInterval))
	}
}

// stopPeriodicSaving останавливает периодическое сохранение
func (s *Server) stopPeriodicSaving() {
	if !s.cfg.IsSyncStore() && s.cfg.StoreInterval > 0 {
		logger.Log.Info("Останавливаем периодическое сохранение...")
		s.storage.StopSavingLoop()
	}
}

// loadStoredData загружает сохраненные данные при старте
func (s *Server) loadStoredData() error {
	if !s.cfg.Restore {
		logger.Log.Info("Загрузка метрик отключена (RESTORE=false)")
		return nil
	}

	// Проверяем существует ли файл
	if _, err := os.Stat(s.cfg.FileStoragePath); err != nil {
		if os.IsNotExist(err) {
			logger.Log.Info("Файл метрик не найден, начинаем с пустого хранилища",
				zap.String("file", s.cfg.FileStoragePath))
			return nil
		}
		logger.Log.Error("Ошибка проверки файла метрик", zap.Error(err))
		return nil // Не критично, продолжаем работу
	}

	// Файл существует - загружаем
	if err := s.storage.LoadFromFile(); err != nil {
		logger.Log.Error("Ошибка загрузки метрик из файла", zap.Error(err))
		return nil // Не критично, продолжаем работу
	}

	logger.Log.Info("Метрики успешно загружены из файла",
		zap.String("file", s.cfg.FileStoragePath))
	return nil
}

// saveFinalMetrics сохраняет метрики при завершении
func (s *Server) saveFinalMetrics() {
	logger.Log.Info("Сохраняем метрики при завершении...")
	if err := s.storage.SaveToFile(); err != nil {
		logger.Log.Error("Ошибка сохранения метрик при завершении", zap.Error(err))
	} else {
		logger.Log.Info("Метрики сохранены при завершении",
			zap.String("file", s.cfg.FileStoragePath))
	}
}

// initializeServices создает сервисы
func (s *Server) initializeServices() {
	s.metricsService = service.New(s.storage)
}

// setupRouter настраивает HTTP роутер
func (s *Server) setupRouter() {
	s.cfg.ApplyGinMode()
	s.router = gin.New()

	// Добавляем middleware
	s.router.Use(gin.Recovery())
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.RequestDecompression())
	s.router.Use(middleware.ResponseCompression())

	// Настраиваем маршруты
	h := handler.New(s.metricsService)
	h.SetupRoutes(s.router)
}

// createHTTPServer создает HTTP сервер
func (s *Server) createHTTPServer() {
	s.httpServer = &http.Server{
		Addr:    s.cfg.Address,
		Handler: s.router,
	}
}
