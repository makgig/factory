package config

import (
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/gin-gonic/gin"
)

const (
	DefaultStoreIntervalsec = 300
)

type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	Loglevel        string `env:"LOG_LEVEL"`
	Ginmod          string `env:"GIN_MODE" envDefault:"release"`
	StoreInterval   time.Duration
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func LoadServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{
		Address:         "localhost:8080",
		Loglevel:        "info",
		StoreInterval:   DefaultStoreIntervalsec * time.Second, // Интервал в секундах по умолчанию
		FileStoragePath: "metrics.json",                        // файл по умолчанию
		Restore:         true,                                  // по умолчанию загружаем данные
		DatabaseDSN:     "",
	}
	address := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	loglevel := flag.String("l", cfg.Loglevel, "уровень логирования")
	storeInterval := flag.Int("i", int(cfg.StoreInterval.Seconds()), "интервал сохранения метрик на диск (секунды)")
	fileStoragePath := flag.String("f", cfg.FileStoragePath, "путь до файла для сохранения метрик")
	restore := flag.Bool("r", cfg.Restore, "загружать ли ранее сохранённые значения при старте")
	dsn := flag.String("d", cfg.DatabaseDSN, "адрес подключения к базе данных (PostgreSQL DSN)")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.Address = *address
	cfg.Loglevel = *loglevel
	cfg.StoreInterval = time.Duration(*storeInterval) * time.Second
	cfg.FileStoragePath = *fileStoragePath
	cfg.Restore = *restore
	cfg.DatabaseDSN = *dsn

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	// Дополнительная обработка для STORE_INTERVAL (поддерживаем int в секундах)
	if storeIntervalStr := os.Getenv("STORE_INTERVAL"); storeIntervalStr != "" {
		if seconds, err := strconv.Atoi(storeIntervalStr); err == nil {
			cfg.StoreInterval = time.Duration(seconds) * time.Second
		}
	}

	// Дополнительная обработка для RESTORE (поддерживаем строки true/false)
	if restoreStr := os.Getenv("RESTORE"); restoreStr != "" {
		if restore, err := strconv.ParseBool(restoreStr); err == nil {
			cfg.Restore = restore
		}
	}

	return cfg, nil
}

func (c *ServerConfig) ApplyGinMode() {
	gin.SetMode(c.Ginmod)
}

// IsSyncStore возвращает true если нужно сохранять данные синхронно (интервал = 0)
func (c *ServerConfig) IsSyncStore() bool {
	return c.StoreInterval == 0
}
