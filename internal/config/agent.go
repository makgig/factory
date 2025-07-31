package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type AgentConfig struct {
	Address        string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func LoadAgentConfig() (*AgentConfig, error) {
	cfg := &AgentConfig{
		Address:        "localhost:8080",
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
	}
	address := flag.String("a", cfg.Address, "адрес эндпоинта HTTP-сервера")
	reportInt := flag.Int("r", int(cfg.ReportInterval.Seconds()), "частота отправки метрик на сервер (секунды)")
	pollInt := flag.Int("p", int(cfg.PollInterval.Seconds()), "частота опроса метрик из runtime (секунды)")

	if !flag.Parsed() {
		flag.Parse()
	}

	cfg.Address = *address
	cfg.ReportInterval = time.Duration(*reportInt) * time.Second
	cfg.PollInterval = time.Duration(*pollInt) * time.Second

	// "github.com/caarlos0/env/v10"
	// Все просто и готово к использованию, но ожидает формат time.Duration (5s) В тестах передается Int (5)

	// if err := env.Parse(cfg); err != nil {
	// 	return nil, err
	// }

	if addr := os.Getenv("ADDRESS"); addr != "" {
		cfg.Address = addr
	}

	if reportStr := os.Getenv("REPORT_INTERVAL"); reportStr != "" {
		if seconds, err := strconv.Atoi(reportStr); err != nil {
			return nil, fmt.Errorf("REPORT_INTERVAL must be integer, got: %s", reportStr)
		} else {
			cfg.ReportInterval = time.Duration(seconds) * time.Second
		}
	}

	if pollStr := os.Getenv("POLL_INTERVAL"); pollStr != "" {
		if seconds, err := strconv.Atoi(pollStr); err != nil {
			return nil, fmt.Errorf("POLL_INTERVAL must be integer, got: %s", pollStr)
		} else {
			cfg.PollInterval = time.Duration(seconds) * time.Second
		}
	}

	return cfg, nil
}

func (c *AgentConfig) ServerURL() string {
	return "http://" + c.Address
}
