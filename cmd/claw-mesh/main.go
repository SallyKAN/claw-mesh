package main

import (
	"fmt"
	"os"

	"github.com/snapek/claw-mesh/internal/config"
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
			fmt.Fprintf(os.Stderr, "starting coordinator on :%d\n", cfg.Coordinator.Port)
			// TODO: start coordinator server (Phase 2)
			return nil
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
			_, err := loadConfig(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "joining mesh at %s\n", args[0])
			// TODO: start node agent (Phase 3)
			return nil
		},
	}
	cmd.Flags().String("name", "", "node display name")
	cmd.Flags().StringSlice("tags", nil, "capability tags")
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
