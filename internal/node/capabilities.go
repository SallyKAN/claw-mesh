package node

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/snapek/claw-mesh/internal/types"
)

// DetectCapabilities gathers the local machine's capabilities.
// extraTags are user-supplied tags from --tags flag.
func DetectCapabilities(extraTags []string) types.Capabilities {
	caps := types.Capabilities{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		GPU:      detectGPU(),
		MemoryGB: detectMemoryGB(),
		Tags:     extraTags,
		Skills:   discoverSkills(),
	}
	return caps
}

// detectGPU checks for GPU availability.
func detectGPU() bool {
	switch runtime.GOOS {
	case "darwin":
		// Apple Silicon always has GPU via Metal.
		return runtime.GOARCH == "arm64"
	case "linux":
		// Check for NVIDIA GPU.
		_, err := exec.LookPath("nvidia-smi")
		return err == nil
	}
	return false
}

// detectMemoryGB returns total system memory in GB.
func detectMemoryGB() int {
	switch runtime.GOOS {
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
		if err != nil {
			return 0
		}
		bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		if err != nil {
			return 0
		}
		return int(bytes / (1 << 30))
	case "linux":
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return 0
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, err := strconv.ParseInt(fields[1], 10, 64)
					if err == nil {
						return int(kb / (1 << 20))
					}
				}
			}
		}
	}
	return 0
}

// discoverSkills looks for known tools on the system.
func discoverSkills() []string {
	var skills []string
	toolMap := map[string]string{
		"docker":       "docker",
		"xcodebuild":  "xcode",
		"python3":     "python",
		"node":        "nodejs",
		"go":          "golang",
		"rustc":       "rust",
		"kubectl":     "kubernetes",
	}
	for bin, skill := range toolMap {
		if _, err := exec.LookPath(bin); err == nil {
			skills = append(skills, skill)
		}
	}

	// Check for OpenClaw gateway config.
	if _, err := findOpenClawConfig(); err == nil {
		skills = append(skills, "openclaw-gateway")
	}

	return skills
}

// findOpenClawConfig searches common locations for openclaw.json.
func findOpenClawConfig() (string, error) {
	candidates := []string{
		"openclaw.json",
		filepath.Join(os.Getenv("HOME"), ".openclaw", "openclaw.json"),
		"/etc/openclaw/openclaw.json",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}
