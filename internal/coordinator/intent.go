package coordinator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

const intentSystemPrompt = `You are a message router for a distributed AI assistant mesh.
Given a user message and a list of available nodes with their capabilities, pick the best node to handle the message.
Consider: OS requirements (darwin for macOS-native tasks, linux for server/container tasks), GPU availability for image generation or ML tasks, tags and skills.
Reply ONLY with a JSON object on a single line: {"node":"<node_name>","reason":"<one short phrase>"}
If the message does not clearly favor any specific node, return: {"node":"","reason":"unclear"}`

// preferredModels lists model IDs in priority order (fastest/cheapest first).
var preferredModels = []string{
	"claude-haiku-4-5-20251001",
	"claude-haiku",
	"claude-sonnet-4-6",
	"claude-sonnet",
	"claude-opus-4-6",
	"claude-opus",
}

// openclawProvider holds the info needed to call an Anthropic-compatible LLM directly.
type openclawProvider struct {
	baseURL string
	apiKey  string
	model   string
}

// LLMClassifier calls an Anthropic-compatible LLM directly using the provider
// configuration from the local OpenClaw config file. No new configuration is
// needed — it reuses what OpenClaw already has.
type LLMClassifier struct {
	provider openclawProvider
	client   *http.Client
}

// ClassifyResult is the routing decision returned by the LLM.
type ClassifyResult struct {
	NodeName string `json:"node"`
	Reason   string `json:"reason"`
}

// newLLMClassifierFromOpenClaw reads the OpenClaw config and builds a classifier
// using the first Anthropic-compatible provider that has models configured.
// openclawCfgPath may be empty, in which case ~/.openclaw/openclaw.json is used.
func newLLMClassifierFromOpenClaw(openclawCfgPath string) *LLMClassifier {
	if openclawCfgPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		openclawCfgPath = filepath.Join(home, ".openclaw", "openclaw.json")
	}

	data, err := os.ReadFile(openclawCfgPath)
	if err != nil {
		log.Printf("intent classifier: cannot read openclaw config %s: %v", openclawCfgPath, err)
		return nil
	}

	provider, err := extractProvider(data)
	if err != nil {
		log.Printf("intent classifier: %v", err)
		return nil
	}

	log.Printf("intent classifier: using %s model %s", provider.baseURL, provider.model)
	return &LLMClassifier{
		provider: *provider,
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// extractProvider parses openclaw.json and returns the best provider+model combo.
func extractProvider(data []byte) (*openclawProvider, error) {
	var cfg struct {
		Models struct {
			Providers map[string]struct {
				API     string `json:"api"`
				APIKey  string `json:"apiKey"`
				BaseURL string `json:"baseUrl"`
				Models  []struct {
					ID string `json:"id"`
				} `json:"models"`
			} `json:"providers"`
		} `json:"models"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse openclaw config: %w", err)
	}

	// Collect all Anthropic-compatible providers that have at least one model.
	type candidate struct {
		baseURL string
		apiKey  string
		model   string
		score   int // lower = better (matches preferredModels order)
	}
	var best *candidate

	for _, p := range cfg.Models.Providers {
		if p.API != "anthropic-messages" || p.APIKey == "" || p.BaseURL == "" {
			continue
		}
		for _, m := range p.Models {
			score := modelScore(m.ID)
			if best == nil || score < best.score {
				best = &candidate{
					baseURL: strings.TrimRight(p.BaseURL, "/"),
					apiKey:  p.APIKey,
					model:   m.ID,
					score:   score,
				}
			}
		}
	}

	if best == nil {
		return nil, fmt.Errorf("no anthropic-compatible provider found in openclaw config")
	}
	return &openclawProvider{baseURL: best.baseURL, apiKey: best.apiKey, model: best.model}, nil
}

// modelScore returns a priority index for a model ID (lower = preferred).
func modelScore(id string) int {
	lower := strings.ToLower(id)
	for i, pref := range preferredModels {
		if strings.Contains(lower, strings.ToLower(pref)) {
			return i
		}
	}
	return len(preferredModels) // unknown models go last
}

// Classify asks the LLM which node should handle the message.
// Returns nil if the call fails or the answer is ambiguous.
func (c *LLMClassifier) Classify(ctx context.Context, content string, nodes []*types.Node) *ClassifyResult {
	userMsg := buildUserMessage(content, nodes)

	// Anthropic Messages API request body.
	reqBody := map[string]interface{}{
		"model":      c.provider.model,
		"max_tokens": 128,
		"system":     intentSystemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": userMsg},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.provider.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.provider.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		log.Printf("intent classify: LLM call failed: %v", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		log.Printf("intent classify: LLM returned %d: %s", resp.StatusCode, body)
		return nil
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil
	}

	// Anthropic Messages API response: content[0].text
	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &anthropicResp); err != nil {
		log.Printf("intent classify: parse response failed: %v", err)
		return nil
	}
	if len(anthropicResp.Content) == 0 {
		return nil
	}

	result := parseClassifyResult(anthropicResp.Content[0].Text)
	if result != nil {
		log.Printf("intent classify: %s → node=%q reason=%q", truncate(content, 30), result.NodeName, result.Reason)
	}
	return result
}

// parseClassifyResult extracts a ClassifyResult from the LLM's text response.
func parseClassifyResult(raw string) *ClassifyResult {
	raw = strings.TrimSpace(raw)
	// Strip markdown code fences if present.
	if idx := strings.Index(raw, "```"); idx != -1 {
		raw = raw[idx+3:]
		if nl := strings.Index(raw, "\n"); nl != -1 {
			raw = raw[nl+1:]
		}
		if end := strings.Index(raw, "```"); end != -1 {
			raw = raw[:end]
		}
		raw = strings.TrimSpace(raw)
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end <= start {
		return nil
	}
	var result ClassifyResult
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return nil
	}
	if result.NodeName == "" {
		return nil
	}
	return &result
}

// buildUserMessage constructs the user-facing prompt listing all online nodes.
func buildUserMessage(content string, nodes []*types.Node) string {
	var sb strings.Builder
	sb.WriteString("Available nodes:\n")
	for _, n := range nodes {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", n.Name, formatNodeCaps(&n.Capabilities)))
	}
	sb.WriteString("\nUser message: ")
	sb.WriteString(content)
	return sb.String()
}

// formatNodeCaps returns a compact one-line capability description.
func formatNodeCaps(c *types.Capabilities) string {
	var parts []string
	if c.OS != "" {
		parts = append(parts, "os="+c.OS)
	}
	if c.Arch != "" {
		parts = append(parts, "arch="+c.Arch)
	}
	if c.GPU {
		parts = append(parts, "gpu=true")
	}
	if c.MemoryGB > 0 {
		parts = append(parts, fmt.Sprintf("mem=%dGB", c.MemoryGB))
	}
	if len(c.Tags) > 0 {
		parts = append(parts, "tags=["+strings.Join(c.Tags, ",")+"]")
	}
	if len(c.Skills) > 0 {
		parts = append(parts, "skills=["+strings.Join(c.Skills, ",")+"]")
	}
	return strings.Join(parts, " ")
}
