package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/justestif/qry/internal/config"
	"github.com/justestif/qry/internal/info"
	"github.com/justestif/qry/internal/router"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func Execute(version string) {
	rootCmd := &cobra.Command{
		Use:     "qry <query>",
		Short:   "A terminal-native, agent-first web search hub",
		Version: version,
		Long: `qry routes search queries through pluggable adapter binaries.
Adapters are external executables registered in ~/.config/qry/config.toml.`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentInfo, _ := cmd.Flags().GetBool("agent-info")

			cfg := &config.Config{}
			if err := viper.Unmarshal(cfg); err != nil {
				return fmt.Errorf("failed to parse config: %w", err)
			}

			// Snapshot raw adapter configs before env expansion so --agent-info
			// can show ${VAR} template strings instead of resolved secret values.
			rawAdapterConfigs := snapshotAdapterConfigs(cfg)
			cfg.ExpandEnv()

			if agentInfo {
				payload := info.Build(version, cfg, viper.ConfigFileUsed(), rawAdapterConfigs)
				out, err := json.MarshalIndent(payload, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to encode agent-info: %w", err)
				}
				fmt.Println(string(out))
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("requires a query argument — run 'qry --agent-info' for usage")
			}
			query := args[0]

			if v := viper.GetString("mode"); v != "" {
				cfg.Routing.Mode = v
			}
			if v := viper.GetString("adapter"); v != "" {
				cfg.Routing.Pool = []string{v}
				cfg.Routing.Fallback = nil
			}
			if v := viper.GetInt("num"); v != 0 {
				cfg.Defaults.Num = v
			}
			if v := viper.GetString("timeout"); v != "" {
				d, err := time.ParseDuration(v)
				if err != nil {
					return fmt.Errorf("invalid --timeout %q: %w", v, err)
				}
				cfg.Defaults.Timeout = d
			}

			cfg.ApplyDefaults()

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			r := router.New(cfg, query)
			output, err := r.Run(context.Background())
			if err != nil {
				var failed router.FailureReporter
				if errors.As(err, &failed) {
					failJSON, _ := json.Marshal(failed.FailureOutput())
					fmt.Fprintln(os.Stderr, string(failJSON))
					os.Exit(1)
				}
				return err
			}

			outJSON, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to encode output: %w", err)
			}
			fmt.Println(string(outJSON))
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.config/qry/config.toml)")
	rootCmd.Flags().BoolP("agent-info", "A", false, "print tool and config info as JSON and exit (no query required)")
	rootCmd.Flags().String("adapter", "", "use a specific adapter, bypassing routing")
	rootCmd.Flags().String("mode", "", "routing mode: first or merge (overrides config)")
	rootCmd.Flags().Int("num", 0, "number of results to return (overrides config)")
	rootCmd.Flags().String("timeout", "", "per-adapter timeout e.g. 5s (overrides config)")

	viper.BindPFlag("adapter", rootCmd.Flags().Lookup("adapter"))
	viper.BindPFlag("mode", rootCmd.Flags().Lookup("mode"))
	viper.BindPFlag("num", rootCmd.Flags().Lookup("num"))
	viper.BindPFlag("timeout", rootCmd.Flags().Lookup("timeout"))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// snapshotAdapterConfigs copies adapter config maps before env expansion.
// This preserves ${VAR} template strings for display in --agent-info output.
func snapshotAdapterConfigs(cfg *config.Config) map[string]map[string]string {
	snapshot := make(map[string]map[string]string, len(cfg.Adapters))
	for name, adapter := range cfg.Adapters {
		if adapter.Config == nil {
			continue
		}
		m := make(map[string]string, len(adapter.Config))
		for k, v := range adapter.Config {
			m[k] = v
		}
		snapshot[name] = m
	}
	return snapshot
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(home + "/.config/qry")
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "error reading config:", err)
			os.Exit(1)
		}
	}
}
