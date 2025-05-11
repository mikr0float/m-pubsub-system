package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config содержит все настройки сервиса
type Config struct {
	GRPCPort         int           `yaml:"grpc_port" env:"GRPC_PORT" env-default:"50051"`
	ShutdownTimeout  time.Duration `yaml:"shutdown_timeout" env:"SHUTDOWN_TIMEOUT" env-default:"10s"`
	LogLevel         string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
	MaxSubscribers   int           `yaml:"max_subscribers" env:"MAX_SUBS" env-default:"1000"`
	MessageQueueSize int           `yaml:"message_queue_size" env:"MSG_QUEUE_SIZE" env-default:"100"`
}

// Load загружает конфигурацию из файла и переменных окружения
func Load(path string) (*Config, error) {
	cfg := &Config{}

	// 1. Сначала пробуем загрузить из YAML файла
	if err := loadFromYAML(path, cfg); err != nil {
		return nil, fmt.Errorf("yaml load failed: %w", err)
	}

	// 2. Переопределяем значения из переменных окружения
	loadFromEnv(cfg)

	// 3. Валидация значений
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// loadFromYAML загружает конфигурацию из YAML файла
func loadFromYAML(path string, cfg *Config) error {
	// Реализация чтения YAML
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл конфига не обязателен
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return err
	}

	return nil
}

// loadFromEnv переопределяет значения из переменных окружения
func loadFromEnv(cfg *Config) {
	// GRPC порт
	if portStr := os.Getenv("GRPC_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
			cfg.GRPCPort = port
		}
	}

	// Таймаут shutdown
	if timeoutStr := os.Getenv("SHUTDOWN_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.ShutdownTimeout = timeout
		}
	}

	// Уровень логирования
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	// Макс. подписчиков
	if maxSubsStr := os.Getenv("MAX_SUBS"); maxSubsStr != "" {
		if maxSubs, err := strconv.Atoi(maxSubsStr); err == nil {
			cfg.MaxSubscribers = maxSubs
		}
	}

	// Размер очереди сообщений
	if queueSizeStr := os.Getenv("MSG_QUEUE_SIZE"); queueSizeStr != "" {
		if queueSize, err := strconv.Atoi(queueSizeStr); err == nil {
			cfg.MessageQueueSize = queueSize
		}
	}
}

// validateConfig проверяет корректность значений конфигурации
func validateConfig(cfg *Config) error {
	// Проверка порта
	if cfg.GRPCPort < 1 || cfg.GRPCPort > 65535 {
		return fmt.Errorf("invalid GRPC port: %d", cfg.GRPCPort)
	}

	// Проверка таймаута
	if cfg.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}

	// Допустимые уровни логирования
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.LogLevel] {
		return fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}

	// Проверка ограничений подписчиков
	if cfg.MaxSubscribers <= 0 {
		return fmt.Errorf("max subscribers must be positive")
	}

	// Проверка размера очереди
	if cfg.MessageQueueSize <= 0 {
		return fmt.Errorf("message queue size must be positive")
	}

	return nil
}
