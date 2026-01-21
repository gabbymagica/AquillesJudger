package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	APIUrl             string
	APIKey             string
	CacheDirectory     string
	CacheFileExtension string
	ExecutionDirectory string
	RunnerBinaryPath   string
	ContainerTimeout   time.Duration
	MaxWorkers         int
	QueueSize          int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		APIUrl:             getEnv("API_URL", "http://localhost:4040/CasoTeste/problemaInterno"),
		APIKey:             getEnv("API_KEY", "token-mega-secreto-que-ninguem-nunca-sabera-#trocarissodepoispraacessardoenv"),
		CacheDirectory:     getEnv("CACHE_DIRECTORY", "../internal/api/cache/"),
		CacheFileExtension: getEnv("CACHE_FILEEXTENSION", "-problem"),
		ExecutionDirectory: getEnv("EXECUTION_DIRECTORY", "../internal/api/cache/executions"),
		RunnerBinaryPath:   getEnv("RUNNER_BINARY_PATH", "../internal/api/binaries/runner"),
	}

	var err error

	seconds, err := strconv.Atoi(getEnv("CONTAINER_TIMEOUT_SECONDS", "600"))
	if err != nil {
		return nil, fmt.Errorf("erro ao ler CONTAINER_TIMEOUT_SECONDS: %w", err)
	}
	cfg.ContainerTimeout = time.Duration(seconds) * time.Second

	cfg.MaxWorkers, err = strconv.Atoi(getEnv("MAX_WORKERS", "3"))
	if err != nil {
		return nil, fmt.Errorf("erro ao ler MAX_WORKERS: %w", err)
	}

	cfg.QueueSize, err = strconv.Atoi(getEnv("QUEUE_SIZE", "10"))
	if err != nil {
		return nil, fmt.Errorf("erro ao ler QUEUE_SIZE: %w", err)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
