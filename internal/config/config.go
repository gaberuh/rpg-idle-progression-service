package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server  ServerConfig
	DB      DBConfig
	Redis   RedisConfig
	Kafka   KafkaConfig
	JWT     JWTConfig
	Worker  WorkerConfig
	Swagger SwaggerConfig
	Otel    OtelConfig
	Log     LogConfig
}

type ServerConfig struct {
	Port string
}

type DBConfig struct {
	URL        string
	MaxConns   int32
	WorkerSema int64
}

type RedisConfig struct {
	Addrs    []string
	Password string
}

type KafkaConfig struct {
	Brokers []string
}

type JWTConfig struct {
	Secret string
}

type WorkerConfig struct {
	HuntInterval  time.Duration
	MaxConcurrent int64
}

type SwaggerConfig struct {
	Enabled bool
}

type OtelConfig struct {
	ServiceName string
}

type LogConfig struct {
	Level slog.Level
}

func Load() (*Config, error) {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		return nil, fmt.Errorf("SERVER_PORT is required")
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL is required")
	}

	maxConns := int32(40)
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("DB_MAX_CONNS invalid: %w", err)
		}
		maxConns = int32(n)
	}

	workerSema := int64(30)
	if v := os.Getenv("WORKER_MAX_CONCURRENT"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("WORKER_MAX_CONCURRENT invalid: %w", err)
		}
		workerSema = n
	}

	redisAddrs := os.Getenv("REDIS_CLUSTER_NODES")
	if redisAddrs == "" {
		return nil, fmt.Errorf("REDIS_CLUSTER_NODES is required")
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		return nil, fmt.Errorf("KAFKA_BROKERS is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	huntInterval := time.Minute
	if v := os.Getenv("HUNT_WORKER_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("HUNT_WORKER_INTERVAL invalid: %w", err)
		}
		huntInterval = d
	}

	logLevel := slog.LevelInfo
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		if err := logLevel.UnmarshalText([]byte(v)); err != nil {
			return nil, fmt.Errorf("LOG_LEVEL inválido %q: use DEBUG, INFO, WARN ou ERROR", v)
		}
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "rpg-idle-progression-service"
	}

	return &Config{
		Server:  ServerConfig{Port: port},
		DB:      DBConfig{URL: dbURL, MaxConns: maxConns, WorkerSema: workerSema},
		Redis:   RedisConfig{Addrs: strings.Split(redisAddrs, ","), Password: os.Getenv("REDIS_PASSWORD")},
		Kafka:   KafkaConfig{Brokers: strings.Split(kafkaBrokers, ",")},
		JWT:     JWTConfig{Secret: jwtSecret},
		Worker:  WorkerConfig{HuntInterval: huntInterval, MaxConcurrent: workerSema},
		Swagger: SwaggerConfig{Enabled: os.Getenv("SWAGGER_ENABLED") == "true"},
		Otel:    OtelConfig{ServiceName: serviceName},
		Log:     LogConfig{Level: logLevel},
	}, nil
}
