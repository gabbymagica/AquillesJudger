package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	APIUrl             string
	CallbackUrl        string
	APIKey             string
	CacheDirectory     string
	CacheFileExtension string
	ExecutionDirectory string
	RunnerBinaryPath   string
<<<<<<< HEAD
	DatabasePath       string
=======
	OnlyLocalCache     bool
>>>>>>> 478b3ab821c2dbd4891f3a3ff245f4f8f5d34585
	ContainerTimeout   time.Duration
	MaxWorkers         int
	QueueSize          int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	// Obtém o diretório do executável para caminhos relativos
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}
	baseDir := filepath.Dir(execPath)

	cfg := &Config{
		APIUrl:             getEnv("API_URL", "http://localhost:4040/CasoTeste/problemaInterno"),
		CallbackUrl:        getEnv("API_CALLBACK_URL", "http://localhost:4040/api/callbacks/judger"),
		APIKey:             getEnv("API_KEY", "token-mega-secreto-que-ninguem-nunca-sabera-#trocarissodepoispraacessardoenv"),
		CacheDirectory:     getEnvPath("CACHE_DIRECTORY", baseDir, "internal/api/cache"),
		CacheFileExtension: getEnv("CACHE_FILEEXTENSION", "-problem"),
		ExecutionDirectory: getEnvPath("EXECUTION_DIRECTORY", baseDir, "internal/api/cache/executions"),
		RunnerBinaryPath:   getEnvPath("RUNNER_BINARY_PATH", baseDir, "internal/api/binaries/runner"),
		DatabasePath:       getEnvPath("DATABASE_PATH", baseDir, "judger.db"),
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

// getEnvPath obtém uma variável de ambiente ou retorna um caminho relativo ao baseDir
func getEnvPath(key, baseDir, relativePath string) string {
	if value, exists := os.LookupEnv(key); exists {
		// Se a variável de ambiente estiver definida, usa ela
		return value
	}
	// Caso contrário, constrói o caminho relativo ao executável
	return filepath.Join(baseDir, relativePath)
}
