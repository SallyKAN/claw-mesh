package node

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// RuntimeKind identifies which AI runtime to use.
type RuntimeKind string

const (
	RuntimeOpenClaw RuntimeKind = "openclaw"
	RuntimeZeroClaw RuntimeKind = "zeroclaw"
	RuntimeNone     RuntimeKind = "none"
)

// RuntimeInfo describes a detected or installed runtime.
type RuntimeInfo struct {
	Kind     RuntimeKind
	Endpoint string // host:port
	Version  string
	Path     string // binary/install path
}

// DetectRuntime checks for locally installed runtimes.
// Returns the first one found in order: OpenClaw, ZeroClaw.
func DetectRuntime() *RuntimeInfo {
	// Check OpenClaw Gateway
	if info := detectOpenClaw(); info != nil {
		return info
	}
	// Check ZeroClaw
	if info := detectZeroClaw(); info != nil {
		return info
	}
	return nil
}

func detectOpenClaw() *RuntimeInfo {
	// Check if openclaw binary exists
	path, err := exec.LookPath("openclaw")
	if err != nil {
		return nil
	}
	// Try to get version
	out, err := exec.Command(path, "--version").CombinedOutput()
	version := "unknown"
	if err == nil {
		version = strings.TrimSpace(string(out))
	}
	return &RuntimeInfo{
		Kind:    RuntimeOpenClaw,
		Version: version,
		Path:    path,
	}
}

func detectZeroClaw() *RuntimeInfo {
	// Check if zeroclaw binary exists
	path, err := exec.LookPath("zeroclaw")
	if err != nil {
		return nil
	}
	out, err := exec.Command(path, "--version").CombinedOutput()
	version := "unknown"
	if err == nil {
		version = strings.TrimSpace(string(out))
	}
	return &RuntimeInfo{
		Kind:    RuntimeZeroClaw,
		Version: version,
		Path:    path,
	}
}

// RecommendRuntime suggests the best runtime for this device.
func RecommendRuntime() RuntimeKind {
	memMB := getSystemMemoryMB()
	hasNode := hasNodeJS()

	// Low memory or no Node.js → ZeroClaw
	if memMB > 0 && memMB < 512 {
		return RuntimeZeroClaw
	}
	if !hasNode {
		// No Node.js, OpenClaw needs it → recommend ZeroClaw
		return RuntimeZeroClaw
	}
	return RuntimeOpenClaw
}

// InstallRuntime downloads and installs the specified runtime.
func InstallRuntime(kind RuntimeKind) error {
	switch kind {
	case RuntimeOpenClaw:
		return installOpenClaw()
	case RuntimeZeroClaw:
		return installZeroClaw()
	default:
		return fmt.Errorf("unknown runtime: %s", kind)
	}
}

func installOpenClaw() error {
	if !hasNodeJS() {
		return fmt.Errorf("OpenClaw requires Node.js (v18+). Install Node.js first, or use --runtime zeroclaw")
	}
	log.Println("installing OpenClaw via npm...")
	cmd := exec.Command("npm", "install", "-g", "openclaw")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install openclaw failed: %w", err)
	}
	log.Println("OpenClaw installed successfully")
	return nil
}

func installZeroClaw() error {
	log.Println("installing ZeroClaw...")

	// Determine platform
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map to ZeroClaw release naming
	var platform string
	switch {
	case goos == "linux" && goarch == "amd64":
		platform = "x86_64-unknown-linux-gnu"
	case goos == "linux" && goarch == "arm64":
		platform = "aarch64-unknown-linux-gnu"
	case goos == "linux" && goarch == "arm":
		platform = "armv7-unknown-linux-gnueabihf"
	case goos == "darwin" && goarch == "arm64":
		platform = "aarch64-apple-darwin"
	case goos == "darwin" && goarch == "amd64":
		platform = "x86_64-apple-darwin"
	default:
		return fmt.Errorf("unsupported platform: %s/%s", goos, goarch)
	}

	// Use bootstrap script from GitHub
	url := fmt.Sprintf("https://github.com/zeroclaw-labs/zeroclaw/releases/latest/download/zeroclaw-%s.tar.gz", platform)
	log.Printf("downloading ZeroClaw from %s", url)

	// Download and extract
	cmd := exec.Command("sh", "-c", fmt.Sprintf(
		`curl -fsSL "%s" | tar xz -C /usr/local/bin/ 2>/dev/null || `+
			`curl -fsSL "%s" | tar xz -C "$HOME/.local/bin/"`,
		url, url))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ZeroClaw install failed: %w. Try manually: curl -fsSL %s | tar xz", err, url)
	}
	log.Println("ZeroClaw installed successfully")
	return nil
}

func hasNodeJS() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

func getSystemMemoryMB() int {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, _ := strconv.Atoi(fields[1])
					return kb / 1024
				}
			}
		}
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0
		}
		bytes, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		return int(bytes / 1024 / 1024)
	}
	return 0
}
