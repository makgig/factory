package config

import (
	"flag"

	"github.com/caarlos0/env/v10"
	"github.com/gin-gonic/gin"
)

type ServerConfig struct {
	Address string `env:"ADDRESS"`
	Ginmod  string `env:"GIN_MODE" envDefault:"release"`
}

func LoadServerConfig() (*ServerConfig, error) {
	cfg := &ServerConfig{
		Address: "localhost:8080",
	}
	address := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.Address = *address

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *ServerConfig) ApplyGinMode() {
	gin.SetMode(c.Ginmod)
}
