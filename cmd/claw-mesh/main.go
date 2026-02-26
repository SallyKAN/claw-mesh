package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/config"
	"github.com/SallyKAN/claw-mesh/internal/coordinator"
	"github.com/SallyKAN/claw-mesh/internal/node"
	"github.com/SallyKAN/claw-mesh/internal/types"
	"github.com/spf13/cobra"
)

var version = "dev"

// NewRootCmd constructs the root command and all subcommands.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "claw-mesh",
		Short: "Multi-Gateway orchestrator for OpenClaw",
		Long:  "One mesh, many claws — orchestrate OpenClaw across machines.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().String("config", "", "config file path (default: ./claw-mesh.yaml)")
	rootCmd.PersistentFlags().String("coordinator", "http://127.0.0.1:9180", "coordinator URL")
	rootCmd.PersistentFlags().String("token", "", "auth token")

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newUpCmd())
	rootCmd.AddCommand(newJoinCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newNodesCmd())
	rootCmd.AddCommand(newSendCmd())
	rootCmd.AddCommand(newRouteCmd())

	return rootCmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("claw-mesh %s\n", version)
		},
	}
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new claw-mesh configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			if cfgPath == "" {
				cfgPath = "claw-mesh.yaml"
			}
			force, _ := cmd.Flags().GetBool("force")

			if !force {
				_, err := os.Stat(cfgPath)
				if err == nil {
					return fmt.Errorf("%s already exists (use --force to overwrite)", cfgPath)
				}
				if !os.IsNotExist(err) {
					return fmt.Errorf("checking %s: %w", cfgPath, err)
				}
			}

			cfg, err := config.Generate()
			if err != nil {
				return err
			}

			if tls, _ := cmd.Flags().GetBool("tls"); tls {
				cfg.TLS.Enabled = true
			}

			if err := cfg.WriteYAML(cfgPath); err != nil {
				return err
			}

			fmt.Printf("Config written to %s\n", cfgPath)
			return nil
		},
	}
	cmd.Flags().Bool("force", false, "overwrite existing config file")
	cmd.Flags().Bool("tls", false, "enable TLS in config (placeholder)")
	return cmd
}

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the coordinator server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			if p, _ := cmd.Flags().GetInt("port"); p != 0 {
				cfg.Coordinator.Port = p
			}

			if ap, _ := cmd.Flags().GetBool("allow-private"); ap {
				cfg.Coordinator.AllowPrivate = true
			}

			if dd, _ := cmd.Flags().GetString("data-dir"); dd != "" {
				cfg.Coordinator.DataDir = dd
			}

			srv := coordinator.NewServer(&cfg.Coordinator)

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() { errCh <- srv.Start() }()

			select {
			case <-ctx.Done():
				fmt.Fprintln(os.Stderr, "shutting down coordinator...")
				return srv.Shutdown(context.Background())
			case err := <-errCh:
				return err
			}
		},
	}
	cmd.Flags().Int("port", 0, "coordinator listen port (default: 9180)")
	cmd.Flags().Bool("allow-private", false, "allow private/loopback IPs for node endpoints")
	cmd.Flags().String("data-dir", "", "data directory for persistent state (default: ~/.claw-mesh)")
	return cmd
}

func newJoinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join <coordinator-url>",
		Short: "Join a mesh as a node",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			coordinatorURL := args[0]
			name, _ := cmd.Flags().GetString("name")
			tags, _ := cmd.Flags().GetStringSlice("tags")
			token := resolveToken(cmd, cfg)
			listen, _ := cmd.Flags().GetString("listen")

			if name == "" {
				name = cfg.Node.Name
			}
			if len(tags) == 0 {
				tags = cfg.Node.Tags
			}

			// Use --endpoint if provided; otherwise derive from listen address.
			endpoint, _ := cmd.Flags().GetString("endpoint")
			if endpoint == "" {
				endpoint = listen
				if endpoint == "" {
					endpoint = ":9121"
				}

				// If endpoint has no host (just :port), detect the outbound IP
				// so the coordinator knows how to reach this node.
				host, port, _ := net.SplitHostPort(endpoint)
				if host == "" {
					if outIP := detectOutboundIP(); outIP != "" {
						endpoint = net.JoinHostPort(outIP, port)
					}
				}
			}

			if name == "" {
				name, _ = os.Hostname()
			}

			agent := node.NewAgent(node.AgentConfig{
				CoordinatorURL:  coordinatorURL,
				Token:           token,
				Name:            name,
				Endpoint:        endpoint,
				Tags:            tags,
				ListenAddr:      listen,
				GatewayEndpoint: resolveGatewayEndpoint(cmd, cfg),
				GatewayToken:    resolveGatewayTokenFlag(cmd, cfg),
				GatewayTimeout:  resolveGatewayTimeout(cmd, cfg),
			})

			fmt.Fprintf(os.Stderr, "joining mesh at %s as %q\n", coordinatorURL, name)

			// Runtime detection and auto-install
			noGw, _ := cmd.Flags().GetBool("no-gateway")
			runtimeFlag, _ := cmd.Flags().GetString("runtime")
			autoInstall, _ := cmd.Flags().GetBool("auto-install")

			if !noGw && resolveGatewayEndpoint(cmd, cfg) == "" {
				// No gateway found, check for runtime
				rt := node.DetectRuntime()
				if rt != nil {
					fmt.Fprintf(os.Stderr, "detected %s runtime (%s) at %s\n", rt.Kind, rt.Version, rt.Path)
				} else {
					recommended := node.RecommendRuntime()
					if runtimeFlag != "" {
						recommended = node.RuntimeKind(runtimeFlag)
					}
					fmt.Fprintf(os.Stderr, "no AI runtime detected. recommended: %s\n", recommended)

					if autoInstall {
						fmt.Fprintf(os.Stderr, "auto-installing %s...\n", recommended)
						if err := node.InstallRuntime(recommended); err != nil {
							fmt.Fprintf(os.Stderr, "WARN: runtime install failed: %v\n", err)
							fmt.Fprintf(os.Stderr, "continuing in echo mode (no AI runtime)\n")
						}
					} else {
						fmt.Fprintf(os.Stderr, "tip: use --auto-install to install %s, or --no-gateway for echo mode\n", recommended)
					}
				}
			}

			if err := agent.StartHandler(); err != nil {
				return fmt.Errorf("starting handler: %w", err)
			}

			if err := agent.Register(); err != nil {
				return err
			}
			agent.StartHeartbeat()

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			<-ctx.Done()

			fmt.Fprintln(os.Stderr, "shutting down node agent...")
			agent.Shutdown()
			return nil
		},
	}
	cmd.Flags().String("name", "", "node display name")
	cmd.Flags().StringSlice("tags", nil, "capability tags")
	cmd.Flags().String("listen", ":9121", "local handler listen address")
	cmd.Flags().String("endpoint", "", "advertised endpoint address (default: auto-detect outbound IP + listen port)")
	cmd.Flags().String("gateway-endpoint", "", "OpenClaw Gateway endpoint (default: auto-discover)")
	cmd.Flags().String("gateway-token", "", "OpenClaw Gateway auth token")
	cmd.Flags().Int("gateway-timeout", 0, "Gateway request timeout in seconds (default: 120)")
	cmd.Flags().Bool("no-gateway", false, "disable gateway auto-discovery (echo mode)")
	cmd.Flags().String("runtime", "", "AI runtime to use: openclaw or zeroclaw (auto-detect if empty)")
	cmd.Flags().Bool("auto-install", false, "auto-install recommended AI runtime if none detected")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show mesh status",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, token := coordFlags(cmd)
			nodes, err := fetchNodes(base, token)
			if err != nil {
				return err
			}

			online, busy, offline := 0, 0, 0
			for _, n := range nodes {
				switch n.Status {
				case types.NodeStatusOnline:
					online++
				case types.NodeStatusBusy:
					busy++
				default:
					offline++
				}
			}

			fmt.Printf("Mesh: %s\n", base)
			fmt.Printf("Nodes: %d total (%d online, %d busy, %d offline)\n",
				len(nodes), online, busy, offline)

			if len(nodes) > 0 {
				fmt.Println()
				printNodesTable(nodes)
			}
			return nil
		},
	}
}

func newNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nodes",
		Short: "List registered nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, token := coordFlags(cmd)
			nodes, err := fetchNodes(base, token)
			if err != nil {
				return err
			}
			if len(nodes) == 0 {
				fmt.Println("No nodes registered.")
				return nil
			}
			printNodesTable(nodes)
			return nil
		},
	}
}

func newSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <message>",
		Short: "Send a message through the mesh",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			base, token := coordFlags(cmd)
			targetNode, _ := cmd.Flags().GetString("node")
			auto, _ := cmd.Flags().GetBool("auto")

			if targetNode == "" && !auto {
				return fmt.Errorf("specify --node <name> or --auto")
			}

			content := args[0]
			payload, _ := json.Marshal(map[string]string{
				"content": content,
				"source":  "cli",
			})

			var url string
			if auto {
				url = base + "/api/v1/route"
			} else {
				// Resolve node name to ID.
				nodeID, err := resolveNodeID(base, token, targetNode)
				if err != nil {
					return err
				}
				url = base + "/api/v1/route/" + nodeID
			}

			req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("sending message: %w", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
			}

			var msgResp types.MessageResponse
			if err := json.Unmarshal(body, &msgResp); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
			fmt.Printf("Message %s routed to node %s\n", msgResp.MessageID, msgResp.NodeID)
			fmt.Printf("Response: %s\n", msgResp.Response)
			return nil
		},
	}
	cmd.Flags().String("node", "", "target node name or ID")
	cmd.Flags().Bool("auto", false, "auto-route based on rules")
	return cmd
}

func newRouteCmd() *cobra.Command {
	routeCmd := &cobra.Command{
		Use:   "route",
		Short: "Manage routing rules",
	}
	routeCmd.AddCommand(newRouteListCmd())
	routeCmd.AddCommand(newRouteAddCmd())
	return routeCmd
}

func newRouteListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List routing rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, token := coordFlags(cmd)
			rules, err := fetchRules(base, token)
			if err != nil {
				return err
			}
			if len(rules) == 0 {
				fmt.Println("No routing rules configured.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tMATCH\tTARGET\tSTRATEGY")
			for _, r := range rules {
				match := describeMatch(&r.Match)
				target := r.Target
				if target == "" {
					target = "-"
				}
				strategy := r.Strategy
				if strategy == "" {
					strategy = "least-busy"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, match, target, strategy)
			}
			w.Flush()
			return nil
		},
	}
}

func newRouteAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a routing rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			base, token := coordFlags(cmd)
			matchStr, _ := cmd.Flags().GetString("match")
			target, _ := cmd.Flags().GetString("target")

			rule := buildRuleFromMatch(matchStr, target)

			payload, _ := json.Marshal(rule)
			req, err := http.NewRequest(http.MethodPost, base+"/api/v1/rules", bytes.NewReader(payload))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("adding rule: %w", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			if resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
			}

			var created types.RoutingRule
			if err := json.Unmarshal(body, &created); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
			fmt.Printf("Rule added: %s\n", created.ID)
			return nil
		},
	}
	cmd.Flags().String("match", "", "match criteria (e.g. 'gpu:true', 'os:linux', 'skill:docker')")
	cmd.Flags().String("target", "", "target node name")
	_ = cmd.MarkFlagRequired("match")
	return cmd
}

// --- helpers ---

func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	cfgFile, _ := cmd.Flags().GetString("config")
	return config.Load(cfgFile)
}

func coordFlags(cmd *cobra.Command) (string, string) {
	base, _ := cmd.Flags().GetString("coordinator")
	token, _ := cmd.Flags().GetString("token")
	if base == "" {
		base = "http://127.0.0.1:9180"
	}
	// Token precedence: flag -> config file -> CLAW_MESH_TOKEN env var.
	if token == "" {
		cfg, err := loadConfig(cmd)
		if err == nil && cfg.Coordinator.Token != "" {
			token = cfg.Coordinator.Token
		}
	}
	if token == "" {
		token = os.Getenv("CLAW_MESH_TOKEN")
	}
	return base, token
}

// resolveToken returns the auth token using precedence: flag -> config -> env.
func resolveToken(cmd *cobra.Command, cfg *config.Config) string {
	token, _ := cmd.Flags().GetString("token")
	if token != "" {
		return token
	}
	if cfg != nil && cfg.Coordinator.Token != "" {
		return cfg.Coordinator.Token
	}
	if env := os.Getenv("CLAW_MESH_TOKEN"); env != "" {
		return env
	}
	return ""
}

// detectOutboundIP finds the preferred outbound IP by dialing a UDP socket.
func detectOutboundIP() string {
	conn, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	return addr.IP.String()
}

func fetchNodes(base, token string) ([]*types.Node, error) {
	req, err := http.NewRequest(http.MethodGet, base+"/api/v1/nodes", nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to coordinator: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coordinator returned %d", resp.StatusCode)
	}
	var nodes []*types.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("decoding nodes: %w", err)
	}
	return nodes, nil
}

func fetchRules(base, token string) ([]*types.RoutingRule, error) {
	req, err := http.NewRequest(http.MethodGet, base+"/api/v1/rules", nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connecting to coordinator: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("coordinator returned %d", resp.StatusCode)
	}
	var rules []*types.RoutingRule
	if err := json.NewDecoder(resp.Body).Decode(&rules); err != nil {
		return nil, fmt.Errorf("decoding rules: %w", err)
	}
	return rules, nil
}

func resolveNodeID(base, token, nameOrID string) (string, error) {
	nodes, err := fetchNodes(base, token)
	if err != nil {
		return "", err
	}
	for _, n := range nodes {
		if n.ID == nameOrID || n.Name == nameOrID {
			return n.ID, nil
		}
	}
	return "", fmt.Errorf("node %q not found", nameOrID)
}

func printNodesTable(nodes []*types.Node) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tENDPOINT\tOS/ARCH\tGPU\tSKILLS")
	for _, n := range nodes {
		gpu := "no"
		if n.Capabilities.GPU {
			gpu = "yes"
		}
		skills := strings.Join(n.Capabilities.Skills, ",")
		if skills == "" {
			skills = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s/%s\t%s\t%s\n",
			n.ID, n.Name, n.Status, n.Endpoint,
			n.Capabilities.OS, n.Capabilities.Arch,
			gpu, skills)
	}
	w.Flush()
}

func describeMatch(mc *types.MatchCriteria) string {
	if mc.Wildcard != nil && *mc.Wildcard {
		return "*"
	}
	var parts []string
	if mc.RequiresGPU != nil && *mc.RequiresGPU {
		parts = append(parts, "gpu:true")
	}
	if mc.RequiresOS != "" {
		parts = append(parts, "os:"+mc.RequiresOS)
	}
	if mc.RequiresSkill != "" {
		parts = append(parts, "skill:"+mc.RequiresSkill)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ",")
}

// resolveGatewayEndpoint returns the gateway endpoint from flag, config, or auto-discovery.
func resolveGatewayEndpoint(cmd *cobra.Command, cfg *config.Config) string {
	if noGw, _ := cmd.Flags().GetBool("no-gateway"); noGw {
		return ""
	}
	if ep, _ := cmd.Flags().GetString("gateway-endpoint"); ep != "" {
		return ep
	}
	if cfg != nil && cfg.Node.Gateway.Endpoint != "" {
		return cfg.Node.Gateway.Endpoint
	}
	// Auto-discover local gateway.
	if cfg == nil || cfg.Node.Gateway.AutoDiscover == nil || *cfg.Node.Gateway.AutoDiscover {
		if info, err := node.DiscoverGateway(); err == nil {
			fmt.Fprintf(os.Stderr, "discovered OpenClaw Gateway at %s\n", info.Endpoint)
			return info.Endpoint
		}
	}
	return ""
}

// resolveGatewayTokenFlag returns the gateway auth token from flag, env, config, or discovery.
func resolveGatewayTokenFlag(cmd *cobra.Command, cfg *config.Config) string {
	cliToken, _ := cmd.Flags().GetString("gateway-token")
	configToken := ""
	if cfg != nil {
		configToken = cfg.Node.Gateway.Token
	}
	// Try discovery for token if not set elsewhere.
	discoveredToken := ""
	if cliToken == "" && configToken == "" {
		if info, err := node.DiscoverGateway(); err == nil {
			discoveredToken = info.Token
		}
	}
	if configToken != "" && discoveredToken == "" {
		discoveredToken = configToken
	}
	return node.ResolveGatewayToken(cliToken, discoveredToken)
}

// resolveGatewayTimeout returns the gateway timeout from flag or config.
func resolveGatewayTimeout(cmd *cobra.Command, cfg *config.Config) int {
	if t, _ := cmd.Flags().GetInt("gateway-timeout"); t > 0 {
		return t
	}
	if cfg != nil && cfg.Node.Gateway.Timeout > 0 {
		return cfg.Node.Gateway.Timeout
	}
	return 120
}

func buildRuleFromMatch(matchStr, target string) types.RoutingRule {
	rule := types.RoutingRule{Target: target}
	for _, part := range strings.Split(matchStr, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "gpu":
			t := kv[1] == "true"
			rule.Match.RequiresGPU = &t
		case "os":
			rule.Match.RequiresOS = kv[1]
		case "skill":
			rule.Match.RequiresSkill = kv[1]
		}
	}
	return rule
}

func main() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
