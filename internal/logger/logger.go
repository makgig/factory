package logger

import (
	"github.com/makgig/factory/internal/config"
	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()

func Initialize(cfg *config.ServerConfig) error {
	lvl, err := zap.ParseAtomicLevel(cfg.Loglevel)
	if err != nil {
		return err
	}
	// создаём новую конфигурацию логера
	zapCfg := zap.NewProductionConfig()
	// устанавливаем уровень
	zapCfg.Level = lvl
	// создаём логер на основе конфигурации
	zl, err := zapCfg.Build()
	if err != nil {
		return err
	}
	// устанавливаем синглтон
	Log = zl
	return nil
}
