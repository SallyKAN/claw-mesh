package node

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
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

	// Check if global npm prefix is writable; if not, use user-local prefix directly.
	globalPrefix := npmGlobalPrefix()
	if globalPrefix != "" && !isDirWritable(globalPrefix) {
		home, _ := os.UserHomeDir()
		prefix := home + "/.local"
		log.Printf("global npm prefix %s not writable, installing to %s ...", globalPrefix, prefix)
		os.MkdirAll(prefix+"/bin", 0755)
		cmd := exec.Command("npm", "install", "-g", "--prefix", prefix, "openclaw")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("npm install openclaw --prefix %s failed: %w", prefix, err)
		}
		log.Printf("OpenClaw installed to %s/bin/openclaw — make sure %s/bin is in your PATH", prefix, prefix)
		return nil
	}

	cmd := exec.Command("npm", "install", "-g", "openclaw")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install openclaw failed: %w", err)
	}
	log.Println("OpenClaw installed successfully")
	return nil
}

// npmGlobalPrefix returns the npm global prefix directory (e.g. /usr/local).
func npmGlobalPrefix() string {
	out, err := exec.Command("npm", "prefix", "-g").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// isDirWritable checks if a directory is writable by the current user.
func isDirWritable(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return false
	}
	// Try creating a temp file to test write access.
	f, err := os.CreateTemp(dir, ".claw-mesh-write-test-*")
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(f.Name())
	return true
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

// RuntimeStartOpts configures how to start the AI runtime (onboard flags).
type RuntimeStartOpts struct {
	Provider string // auth-choice: anthropic-api-key, openai-api-key, custom-api-key, etc.
	APIKey   string // provider API key
	BaseURL  string // custom base URL (for custom provider)
	Model    string // custom model ID
}

// StartRuntime starts the gateway for the given runtime kind.
// For OpenClaw it runs `openclaw onboard --non-interactive`.
// For ZeroClaw it runs `zeroclaw serve` in the background.
// If the gateway is already running, it returns immediately.
func StartRuntime(kind RuntimeKind, opts RuntimeStartOpts) error {
	switch kind {
	case RuntimeOpenClaw:
		return startOpenClaw(opts)
	case RuntimeZeroClaw:
		return startZeroClaw()
	default:
		return fmt.Errorf("unknown runtime: %s", kind)
	}
}

func startOpenClaw(opts RuntimeStartOpts) error {
	// Already running? Nothing to do.
	if verifyGatewayRunning("127.0.0.1:18789") {
		log.Println("OpenClaw Gateway already running on :18789")
		return nil
	}

	// Find the openclaw binary.
	binPath, err := findOpenClawBinary()
	if err != nil {
		return err
	}

	// Resolve provider + key from opts or environment.
	provider, keyFlag, keyVal, err := resolveProviderOpts(opts)
	if err != nil {
		return err
	}

	// Build onboard command.
	args := []string{"onboard", "--non-interactive", "--accept-risk", "--install-daemon"}
	if provider != "" {
		args = append(args, "--auth-choice", provider)
	}
	if keyFlag != "" && keyVal != "" {
		args = append(args, "--"+keyFlag, keyVal)
	}
	if opts.BaseURL != "" {
		args = append(args, "--custom-base-url", opts.BaseURL)
		// Determine compatibility mode from provider hint or default to anthropic.
		compat := "anthropic"
		if opts.Provider == "openai" || opts.Provider == "custom" {
			compat = "openai"
		}
		args = append(args, "--custom-compatibility", compat)
		// custom-api-key requires a model ID; provide a sensible default.
		if opts.Model == "" {
			if compat == "anthropic" {
				opts.Model = "claude-sonnet-4-20250514"
			} else {
				opts.Model = "gpt-4o"
			}
		}
	}
	if opts.Model != "" {
		args = append(args, "--custom-model-id", opts.Model)
	}

	log.Printf("starting OpenClaw: %s %s", binPath, strings.Join(args, " "))
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("openclaw onboard failed: %w", err)
	}

	// Wait for gateway to become reachable.
	return waitForGateway("127.0.0.1:18789", 30*time.Second)
}

func startZeroClaw() error {
	if verifyGatewayRunning("127.0.0.1:18789") {
		log.Println("ZeroClaw Gateway already running on :18789")
		return nil
	}

	binPath, err := exec.LookPath("zeroclaw")
	if err != nil {
		home, _ := os.UserHomeDir()
		candidate := filepath.Join(home, ".local", "bin", "zeroclaw")
		if _, serr := os.Stat(candidate); serr == nil {
			binPath = candidate
		} else {
			return fmt.Errorf("zeroclaw binary not found")
		}
	}

	log.Printf("starting ZeroClaw: %s serve", binPath)
	cmd := exec.Command(binPath, "serve")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("zeroclaw serve failed: %w", err)
	}

	return waitForGateway("127.0.0.1:18789", 30*time.Second)
}

// findOpenClawBinary locates the openclaw binary in PATH or common locations.
func findOpenClawBinary() (string, error) {
	if p, err := exec.LookPath("openclaw"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "openclaw"),
		"/usr/local/bin/openclaw",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("openclaw binary not found in PATH or common locations")
}

// resolveProviderOpts determines the provider and API key from explicit opts or env vars.
func resolveProviderOpts(opts RuntimeStartOpts) (provider, keyFlag, keyVal string, err error) {
	// If a custom base URL is provided, always use the custom provider path.
	if opts.BaseURL != "" {
		key := opts.APIKey
		if key == "" {
			// Try env vars as fallback for the key value.
			key = os.Getenv("ANTHROPIC_API_KEY")
			if key == "" {
				key = os.Getenv("OPENAI_API_KEY")
			}
		}
		if key == "" {
			return "", "", "", fmt.Errorf("--api-base requires an API key: pass --api-key or set ANTHROPIC_API_KEY / OPENAI_API_KEY")
		}
		return "custom-api-key", "custom-api-key", key, nil
	}

	if opts.APIKey != "" {
		// Explicit key provided — determine provider.
		switch {
		case opts.Provider == "custom":
			return "custom-api-key", "custom-api-key", opts.APIKey, nil
		case opts.Provider == "openai":
			return "openai-api-key", "openai-api-key", opts.APIKey, nil
		default: // anthropic or empty
			return "apiKey", "anthropic-api-key", opts.APIKey, nil
		}
	}

	// No explicit key — try environment variables.
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		return "apiKey", "anthropic-api-key", v, nil
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		return "openai-api-key", "openai-api-key", v, nil
	}

	return "", "", "", fmt.Errorf("no API key provided: pass --api-key or set ANTHROPIC_API_KEY / OPENAI_API_KEY")
}

// waitForGateway polls the endpoint until it's reachable or timeout expires.
func waitForGateway(endpoint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if verifyGatewayRunning(endpoint) {
			log.Printf("gateway reachable at %s", endpoint)
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("gateway at %s not reachable after %s", endpoint, timeout)
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
