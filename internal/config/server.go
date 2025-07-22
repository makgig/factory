package config

import (
	"flag"

	"github.com/caarlos0/env/v10"
	"github.com/gin-gonic/gin"
)

type ServerConfig struct {
	Address  string `env:"ADDRESS"`
	Loglevel string `env:"LOG_LEVEL"`
	Ginmod   string `env:"GIN_MODE" envDefault:"release"`
}

func LoadServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{
		Address:  "localhost:8080",
		Loglevel: "info",
	}
	address := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	loglevel := flag.String("l", cfg.Loglevel, "уровень логирования")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.Address = *address
	cfg.Loglevel = *loglevel

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *ServerConfig) ApplyGinMode() {
	gin.SetMode(c.Ginmod)
}
