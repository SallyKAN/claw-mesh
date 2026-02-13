package node

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GatewayInfo holds discovered OpenClaw Gateway details.
type GatewayInfo struct {
	Endpoint string `json:"endpoint"`
	Version  string `json:"version"`
}

// DiscoverGateway attempts to find a running local OpenClaw Gateway.
// It checks the config file first, then falls back to process detection.
func DiscoverGateway() (*GatewayInfo, error) {
	// Try config-based discovery first.
	if info, err := discoverFromConfig(); err == nil {
		return info, nil
	}

	// Fall back to process-based discovery.
	if info, err := discoverFromProcess(); err == nil {
		return info, nil
	}

	return nil, fmt.Errorf("no local OpenClaw Gateway found")
}

// discoverFromConfig reads the openclaw.json config to find the gateway endpoint.
func discoverFromConfig() (*GatewayInfo, error) {
	configPath, err := findOpenClawConfig()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg struct {
		Gateway struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		} `json:"gateway"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	host := cfg.Gateway.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Gateway.Port
	if port == 0 {
		port = 9120
	}

	return &GatewayInfo{
		Endpoint: fmt.Sprintf("%s:%d", host, port),
		Version:  cfg.Version,
	}, nil
}

// discoverFromProcess looks for a running openclaw process.
func discoverFromProcess() (*GatewayInfo, error) {
	// Look for openclaw binary.
	path, err := exec.LookPath("openclaw")
	if err != nil {
		return nil, fmt.Errorf("openclaw binary not found: %w", err)
	}

	// Try to get version.
	var version string
	out, err := exec.Command(path, "version").Output()
	if err == nil {
		version = strings.TrimSpace(string(out))
	}

	// Check common data dirs for a running gateway port file.
	portFiles := []string{
		filepath.Join(os.Getenv("HOME"), ".openclaw", "gateway.port"),
		"/tmp/openclaw-gateway.port",
	}
	for _, pf := range portFiles {
		data, err := os.ReadFile(pf)
		if err == nil {
			port := strings.TrimSpace(string(data))
			return &GatewayInfo{
				Endpoint: "127.0.0.1:" + port,
				Version:  version,
			}, nil
		}
	}

	// Default endpoint.
	return &GatewayInfo{
		Endpoint: "127.0.0.1:9120",
		Version:  version,
	}, nil
}
