package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"

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
	db             *pgxpool.Pool
	openPool       openPoolFunc
	pingPool       pingPoolFunc
	newMigrator    newMigratorFunc
	closePool      func(*pgxpool.Pool)
}

// New создает новый экземпляр сервера
func New(cfg *config.ServerConfig) *Server {
	return &Server{
		cfg:         cfg,
		openPool:    openPoolDefault,
		pingPool:    pingPoolDefault,
		newMigrator: newMigratorDefault,
		closePool: func(p *pgxpool.Pool) {
			if p != nil {
				p.Close()
			}
		},
	}
}

// Initialize инициализирует все компоненты сервера
func (s *Server) Initialize() error {
	// Инициализируем логер
	if err := s.initializeLogger(); err != nil {
		return err
	}

	// Инициализируем базу данных
	if err := s.initializeDatabase(); err != nil {
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

	// корректно закрываем пул БД
	if s.db != nil {
		s.closePool(s.db)
		s.db = nil
	}

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

// initializeDatabase настраивает подключение к базе данных
func (s *Server) initializeDatabase() error {
	if s.cfg.DatabaseDSN == "" {
		logger.Log.Info("DATABASE_DSN пуст — подключение к БД пропущено")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := s.openPool(ctx, s.cfg.DatabaseDSN)
	if err != nil {
		logger.Log.Error("не удалось подключиться к БД: %v", zap.Error(err))
		return err
	}

	// Проверка соединения
	if err := s.pingPool(ctx, pool); err != nil {
		logger.Log.Error("БД недоступна: %v", zap.Error(err))
		s.closePool(pool)
		return err
	}

	// Инициализация и запуск миграций
	m, err := s.newMigrator(s.cfg.DatabaseDSN)
	if err != nil {
		logger.Log.Error(
			"не удалось инициализировать миграции",
			zap.Error(err),
			zap.String("hint", "проверьте, что папка 'migrations/' существует и содержит SQL-файлы, а DATABASE_DSN корректен"),
		)
		s.closePool(pool)
		return err
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			logger.Log.Info("Нет новых миграций — структура БД уже актуальна")
		} else {
			logger.Log.Error("ошибка применения миграций: %v", zap.Error(err))
			s.closePool(pool)
			return err
		}
	} else {
		logger.Log.Info("Миграции успешно применены")
	}

	s.db = pool
	logger.Log.Info("Подключение к БД установлено (pgxpool)")
	return nil
}

// initializeStorage настраивает хранилище метрик
func (s *Server) initializeStorage() error {
	// если БД инициализирована — используем её
	if s.db != nil {
		s.storage = repository.NewPostgres(s.db)
		logger.Log.Info("Используется хранилище PostgreSQL")
		return nil
	}

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
		// Убеждаемся, что директория для сохранения существует
		if err := s.ensureStorageDirectory(); err != nil {
			logger.Log.Error("Ошибка создания директории для периодического сохранения", zap.Error(err))
			return
		}

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

	// Если путь к файлу не указан, нечего загружать
	if s.cfg.FileStoragePath == "" {
		logger.Log.Info("Путь к файлу метрик не указан, начинаем с пустого хранилища")
		return nil
	}

	// Создаем директорию для файла метрик, если она не существует
	if err := s.ensureStorageDirectory(); err != nil {
		logger.Log.Error("Ошибка создания директории для файла метрик", zap.Error(err))
		return nil // Не критично, продолжаем работу
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

// ensureStorageDirectory создает директорию для файла метрик, если она не существует
func (s *Server) ensureStorageDirectory() error {
	if s.cfg.FileStoragePath == "" {
		return nil
	}

	dir := filepath.Dir(s.cfg.FileStoragePath)

	// Если директория это текущая папка ".", то создавать нечего
	if dir == "." {
		return nil
	}

	// Создаем все необходимые директории
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	logger.Log.Debug("Директория для файла метрик создана",
		zap.String("directory", dir))
	return nil
}

// saveFinalMetrics сохраняет метрики при завершении
func (s *Server) saveFinalMetrics() {
	if s.cfg.FileStoragePath == "" {
		logger.Log.Info("Путь к файлу не указан, пропускаем сохранение при завершении")
		return
	}

	// Убеждаемся, что директория существует перед финальным сохранением
	if err := s.ensureStorageDirectory(); err != nil {
		logger.Log.Error("Ошибка создания директории для финального сохранения", zap.Error(err))
		return
	}

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

	// health маршруты
	health := handler.NewHealth(s.db)
	health.SetupRoutes(s.router)

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
