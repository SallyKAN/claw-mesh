package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// TLSConfig holds TLS settings (placeholder for future use).
type TLSConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	CertFile string `json:"cert_file" yaml:"cert_file" mapstructure:"cert_file"`
	KeyFile  string `json:"key_file" yaml:"key_file" mapstructure:"key_file"`
}

// Config holds the full claw-mesh configuration.
type Config struct {
	Coordinator CoordinatorConfig `json:"coordinator" yaml:"coordinator" mapstructure:"coordinator"`
	Node        NodeConfig        `json:"node" yaml:"node" mapstructure:"node"`
	TLS         TLSConfig         `json:"tls" yaml:"tls" mapstructure:"tls"`
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
	Endpoint     string `json:"endpoint,omitempty" yaml:"endpoint,omitempty" mapstructure:"endpoint"`
	Token        string `json:"token,omitempty" yaml:"token,omitempty" mapstructure:"token"`
	Timeout      int    `json:"timeout,omitempty" yaml:"timeout,omitempty" mapstructure:"timeout"`
	AutoDiscover *bool  `json:"auto_discover,omitempty" yaml:"auto_discover,omitempty" mapstructure:"auto_discover"`
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

// Generate returns a default Config with a random 32-char hex token.
func Generate() (*Config, error) {
	token, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	hostname, _ := os.Hostname()

	return &Config{
		Coordinator: CoordinatorConfig{
			Port:         9180,
			Token:        token,
			AllowPrivate: true,
		},
		Node: NodeConfig{
			Name: hostname,
			Tags: []string{},
		},
		TLS: TLSConfig{
			Enabled: false,
		},
	}, nil
}

// WriteYAML marshals the config to YAML and writes it to path.
func (c *Config) WriteYAML(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
