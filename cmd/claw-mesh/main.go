package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/snapek/claw-mesh/internal/config"
	"github.com/snapek/claw-mesh/internal/coordinator"
	"github.com/snapek/claw-mesh/internal/node"
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

	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newUpCmd())
	rootCmd.AddCommand(newJoinCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newNodesCmd())

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

func newUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start the coordinator server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd)
			if err != nil {
				return err
			}

			// Override port from flag if set.
			if p, _ := cmd.Flags().GetInt("port"); p != 0 {
				cfg.Coordinator.Port = p
			}

			srv := coordinator.NewServer(&cfg.Coordinator)

			// Graceful shutdown on SIGINT/SIGTERM.
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
			token, _ := cmd.Flags().GetString("token")

			// Flags override config values.
			if name == "" {
				name = cfg.Node.Name
			}
			if len(tags) == 0 {
				tags = cfg.Node.Tags
			}
			if token == "" {
				token = cfg.Coordinator.Token
			}

			// Auto-detect endpoint if not configured.
			endpoint := cfg.Node.Endpoint
			if endpoint == "" {
				if gw, err := node.DiscoverGateway(); err == nil {
					endpoint = gw.Endpoint
				} else {
					endpoint = "127.0.0.1:9120"
				}
			}

			// Default name to hostname.
			if name == "" {
				name, _ = os.Hostname()
			}

			agent := node.NewAgent(node.AgentConfig{
				CoordinatorURL: coordinatorURL,
				Token:          token,
				Name:           name,
				Endpoint:       endpoint,
				Tags:           tags,
			})

			fmt.Fprintf(os.Stderr, "joining mesh at %s as %q\n", coordinatorURL, name)

			if err := agent.Register(); err != nil {
				return err
			}
			agent.StartHeartbeat()

			// Block until SIGINT/SIGTERM.
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
	cmd.Flags().String("token", "", "coordinator auth token")
	return cmd
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show mesh status",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: query coordinator API
			fmt.Println("mesh status: not yet implemented")
			return nil
		},
	}
}

func newNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nodes",
		Short: "List registered nodes",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: query coordinator API
			fmt.Println("nodes: not yet implemented")
			return nil
		},
	}
}

// loadConfig loads configuration, respecting the --config flag.
func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	cfgFile, _ := cmd.Flags().GetString("config")
	return config.Load(cfgFile)
}

func main() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
