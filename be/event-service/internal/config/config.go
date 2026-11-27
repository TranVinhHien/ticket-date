package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App         AppMetadataConfig `mapstructure:"app"`
	HttpServer  HttpServerConfig  `mapstructure:"http_server"`
	Logger      LoggerConfig      `mapstructure:"logger"`
	CockroachDB CockroachConfig   `mapstructure:"cockroach_db"`
	Redis       RedisConfig       `mapstructure:"redis"`
	Token       TokenConfig       `mapstructure:"token"`
}

type AppMetadataConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

type HttpServerConfig struct {
	Host         string   `mapstructure:"host"`
	Port         int      `mapstructure:"port"`
	GRPCPort     int      `mapstructure:"grpc_port"`
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
	AllowHeaders []string `mapstructure:"allow_headers"`
	Mode         string   `mapstructure:"mode"`
	LogLevel     string   `mapstructure:"log_level"`
}

type LoggerConfig struct {
	Level string `mapstructure:"level"`
}

type CockroachConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"db_name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	LogLevel        string        `mapstructure:"log_level"`
}

type TokenConfig struct {
	Secret string `mapstructure:"secret"`
}

type RedisConfig struct {
	Host             string        `mapstructure:"host"`
	Port             int           `mapstructure:"port"`
	Password         string        `mapstructure:"password"`
	DB               int           `mapstructure:"db"`
	PoolSize         int           `mapstructure:"pool_size"`
	MinIdleConns     int           `mapstructure:"min_idle_conns"`
	DialTimeout      time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout"`
	CacheTTL         time.Duration `mapstructure:"cache_ttl"`
	NegativeCacheTTL time.Duration `mapstructure:"negative_cache_ttl"`
}

var App Config

// ResolveConfigFile finds the event service configuration file path.
func ResolveConfigFile(customPath string) string {
	if customPath != "" {
		return customPath
	}
	if envPath := os.Getenv("CONFIG_FILE"); envPath != "" {
		return envPath
	}
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		return envPath
	}
	if envPath := os.Getenv("APP_CONFIG"); envPath != "" {
		return envPath
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = os.Getenv("ENV")
	}
	if appEnv == "" {
		appEnv = "development"
	}

	candidates := []string{
		fmt.Sprintf("app.%s.yaml", appEnv),
		fmt.Sprintf("./config/app.%s.yaml", appEnv),
		"app.yaml",
		"./config/app.yaml",
		"app.example.yaml",
		fmt.Sprintf("./event-service/app.%s.yaml", appEnv),
		fmt.Sprintf("../app.%s.yaml", appEnv),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return candidates[0]
}

// LoadConfig loads the event service configuration from file and environment variables.
func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	targetPath := ResolveConfigFile(path)
	v.SetConfigFile(targetPath)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Default values
	v.SetDefault("app.name", "event-service")
	v.SetDefault("app.version", "v1.0.0")
	v.SetDefault("http_server.host", "0.0.0.0")
	v.SetDefault("http_server.mode", "debug")
	v.SetDefault("http_server.log_level", "info")
	v.SetDefault("cockroach_db.host", "localhost")
	v.SetDefault("cockroach_db.port", 5432)
	v.SetDefault("cockroach_db.user", "postgres")
	v.SetDefault("cockroach_db.password", "postgres")
	v.SetDefault("cockroach_db.db_name", "ticket")
	v.SetDefault("cockroach_db.ssl_mode", "disable")
	v.SetDefault("cockroach_db.max_open_conns", 25)
	v.SetDefault("cockroach_db.max_idle_conns", 5)
	v.SetDefault("cockroach_db.conn_max_lifetime", time.Hour)
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 20)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.cache_ttl", 5*time.Minute)
	v.SetDefault("redis.negative_cache_ttl", 1*time.Minute)
	v.SetDefault("token.secret", "change-me-to-a-long-random-secret")

	// Bind environment variables
	_ = v.BindEnv("http_server.port", "PORT", "SERVER_PORT", "HTTP_SERVER_PORT", "EVENT_SERVICE_PORT")
	_ = v.BindEnv("http_server.host", "HOST", "SERVER_HOST", "HTTP_SERVER_HOST")
	_ = v.BindEnv("http_server.mode", "GIN_MODE", "HTTP_SERVER_MODE")
	_ = v.BindEnv("cockroach_db.host", "DB_HOST", "COCKROACH_DB_HOST", "DATABASE_HOST", "POSTGRES_HOST")
	_ = v.BindEnv("cockroach_db.port", "DB_PORT", "COCKROACH_DB_PORT", "DATABASE_PORT", "POSTGRES_PORT")
	_ = v.BindEnv("cockroach_db.user", "DB_USER", "COCKROACH_DB_USER", "DATABASE_USER", "POSTGRES_USER")
	_ = v.BindEnv("cockroach_db.password", "DB_PASSWORD", "COCKROACH_DB_PASSWORD", "DATABASE_PASSWORD", "POSTGRES_PASSWORD")
	_ = v.BindEnv("cockroach_db.db_name", "DB_NAME", "COCKROACH_DB_NAME", "DATABASE_NAME", "POSTGRES_DB")
	_ = v.BindEnv("cockroach_db.ssl_mode", "DB_SSLMODE", "COCKROACH_DB_SSLMODE")
	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.port", "REDIS_PORT")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("redis.db", "REDIS_DB")
	_ = v.BindEnv("token.secret", "JWT_SECRET", "TOKEN_SECRET")

	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) && !strings.Contains(err.Error(), "no such file") {
			return nil, fmt.Errorf("failed to read event service config file '%s': %w", targetPath, err)
		}
	}

	var conf Config
	if err := v.Unmarshal(&conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event service config: %w", err)
	}

	return &conf, nil
}

// Load initializes the global App config using automatic path resolution.
func Load() error {
	conf, err := LoadConfig("")
	if err != nil {
		return err
	}
	App = *conf
	return nil
}
