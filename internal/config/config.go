package config

import (
	"flag"
	"time"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	Address        string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
}

func LoadServerConfig() (*Config, error) {
	cfg := &Config{
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

func LoadAgentConfig() (*Config, error) {
	cfg := &Config{
		Address:        "localhost:8080",
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
	}
	address := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	reportInt := flag.Duration("r", cfg.ReportInterval, "частота отправки метрик на сервер (секунды)")
	pollInt := flag.Duration("p", cfg.PollInterval, "частота опроса метрик из runtime (секунды)")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.Address = *address
	cfg.ReportInterval = *reportInt
	cfg.PollInterval = *pollInt

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) ServerUrl() string {
	return "http://" + c.Address
}
