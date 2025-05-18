package config

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/playconomy/wallet-service/internal/utils"
	
	"github.com/spf13/viper"
)

type Config struct {
	Server        ServerConfig        `validate:"required"`
	Database      DatabaseConfig      `validate:"required"`
	App           AppConfig           `validate:"required"`
	Observability ObservabilityConfig `validate:"required"`
}

type ServerConfig struct {
	Host string `validate:"required"`
	Port int    `validate:"required,gte=1,lte=65535"`
}

type DatabaseConfig struct {
	Host     string `validate:"required"`
	Port     int    `validate:"required,gte=1,lte=65535"`
	User     string `validate:"required"`
	Password string `validate:"required"`
	DBName   string `validate:"required"`
	SSLMode  string `validate:"required,oneof=disable enable verify-ca verify-full"`
}

type AppConfig struct {
	Name     string `validate:"required"`
	Env      string `validate:"required,oneof=development staging production"`
	LogLevel string `validate:"required,oneof=debug info warn error"`
	Version  string `validate:"required"`
}

type ObservabilityConfig struct {
	Tracing TracingConfig `validate:"required"`
	Metrics MetricsConfig `validate:"required"`
}

type TracingConfig struct {
	Enabled        bool    `validate:"required"`
	Endpoint       string  `validate:"required_if=Enabled true"`
	SamplingRatio  float64 `validate:"required,gte=0,lte=1"`
}

type MetricsConfig struct {
	Enabled bool `validate:"required"`
}

// LoadConfig loads configuration from environment file and environment variables
func LoadConfig() (*Config, error) {
	// Get the project root directory
	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "../..")

	viper.SetConfigName("default")
	viper.SetConfigType("env")
	viper.AddConfigPath(filepath.Join(projectRoot, "profiles"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Set default values
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()

	var config Config
	config.Server = ServerConfig{
		Host: viper.GetString("SERVER_HOST"),
		Port: viper.GetInt("SERVER_PORT"),
	}

	config.Database = DatabaseConfig{
		Host:     viper.GetString("DB_HOST"),
		Port:     viper.GetInt("DB_PORT"),
		User:     viper.GetString("DB_USER"),
		Password: viper.GetString("DB_PASSWORD"),
		DBName:   viper.GetString("DB_NAME"),
		SSLMode:  viper.GetString("DB_SSL_MODE"),
	}

	config.App = AppConfig{
		Name:     viper.GetString("APP_NAME"),
		Env:      viper.GetString("APP_ENV"),
		LogLevel: viper.GetString("LOG_LEVEL"),
		Version:  viper.GetString("APP_VERSION"),
	}

	config.Observability = ObservabilityConfig{
		Tracing: TracingConfig{
			Enabled:       viper.GetBool("TRACING_ENABLED"),
			Endpoint:      viper.GetString("TRACING_ENDPOINT"),
			SamplingRatio: viper.GetFloat64("TRACING_SAMPLING_RATIO"),
		},
		Metrics: MetricsConfig{
			Enabled: viper.GetBool("METRICS_ENABLED"),
		},
	}

	// Validate config
	if err := utils.ValidateStruct(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("SERVER_HOST", "localhost")
	viper.SetDefault("SERVER_PORT", 3000)

	// Database defaults
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "postgres")
	viper.SetDefault("DB_NAME", "wallet_db")
	viper.SetDefault("DB_SSL_MODE", "disable")

	// App defaults
	viper.SetDefault("APP_NAME", "wallet-service")
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("APP_VERSION", "0.1.0")
	
	// Observability defaults
	viper.SetDefault("TRACING_ENABLED", false)
	viper.SetDefault("TRACING_ENDPOINT", "localhost:4317")
	viper.SetDefault("TRACING_SAMPLING_RATIO", 0.1)
	viper.SetDefault("METRICS_ENABLED", true)
}

// GetDSN returns database connection string
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
