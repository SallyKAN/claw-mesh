package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the full claw-mesh configuration.
type Config struct {
	Coordinator CoordinatorConfig `json:"coordinator" yaml:"coordinator" mapstructure:"coordinator"`
	Node        NodeConfig        `json:"node" yaml:"node" mapstructure:"node"`
}

// CoordinatorConfig holds coordinator-specific settings.
type CoordinatorConfig struct {
	Port         int    `json:"port" yaml:"port" mapstructure:"port"`
	Token        string `json:"token" yaml:"token" mapstructure:"token"`
	AllowPrivate bool   `json:"allow_private" yaml:"allow_private" mapstructure:"allow_private"`
}

// NodeConfig holds node agent settings.
type NodeConfig struct {
	Name     string        `json:"name" yaml:"name" mapstructure:"name"`
	Tags     []string      `json:"tags" yaml:"tags" mapstructure:"tags"`
	Endpoint string        `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	Gateway  GatewayConfig `json:"gateway" yaml:"gateway" mapstructure:"gateway"`
}

// GatewayConfig holds OpenClaw Gateway connection settings.
type GatewayConfig struct {
	Endpoint     string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
	Token        string `json:"token" yaml:"token" mapstructure:"token"`
	Timeout      int    `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	AutoDiscover bool   `json:"auto_discover" yaml:"auto_discover" mapstructure:"auto_discover"`
}

// Load reads configuration from file and environment.
// If cfgFile is non-empty, it is used as the explicit config path.
func Load(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("claw-mesh")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.claw-mesh")
		viper.AddConfigPath("/etc/claw-mesh")
	}

	viper.SetDefault("coordinator.port", 9180)
	viper.SetDefault("node.gateway.auto_discover", true)
	viper.SetDefault("node.gateway.timeout", 120)

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
