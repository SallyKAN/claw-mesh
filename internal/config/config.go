package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the full claw-mesh configuration.
type Config struct {
	Coordinator CoordinatorConfig `mapstructure:"coordinator"`
	Node        NodeConfig        `mapstructure:"node"`
}

// CoordinatorConfig holds coordinator-specific settings.
type CoordinatorConfig struct {
	Port  int    `mapstructure:"port"`
	Token string `mapstructure:"token"`
}

// NodeConfig holds node agent settings.
type NodeConfig struct {
	Name     string   `mapstructure:"name"`
	Tags     []string `mapstructure:"tags"`
	Endpoint string   `mapstructure:"endpoint"`
}

// Load reads configuration from file and environment.
func Load() (*Config, error) {
	viper.SetConfigName("claw-mesh")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.claw-mesh")
	viper.AddConfigPath("/etc/claw-mesh")

	viper.SetDefault("coordinator.port", 9180)

	viper.SetEnvPrefix("CLAW_MESH")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}
