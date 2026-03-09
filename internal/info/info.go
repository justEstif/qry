// Package info builds the agent-info payload for qry --agent-info.
package info

import (
	"os"
	"time"

	"github.com/justestif/qry/internal/config"
)

// AgentInfo is the top-level structure emitted by qry --agent-info.
type AgentInfo struct {
	Tool         ToolInfo              `json:"tool"`
	RoutingModes map[string]string     `json:"routing_modes"`
	Config       *ConfigInfo           `json:"config"`
}

// ToolInfo describes the binary itself.
type ToolInfo struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Usage       string          `json:"usage"`
	Flags       map[string]Flag `json:"flags"`
}

// Flag describes a single CLI flag.
type Flag struct {
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
}

// ConfigInfo mirrors the resolved config, safe for agent consumption.
// Adapter config map values are shown as-is (template strings like ${VAR},
// not resolved secrets).
type ConfigInfo struct {
	Source   string                   `json:"source"`
	Defaults DefaultsInfo             `json:"defaults"`
	Routing  RoutingInfo              `json:"routing"`
	Adapters map[string]AdapterInfo   `json:"adapters"`
}

// DefaultsInfo holds the global defaults.
type DefaultsInfo struct {
	Num     int    `json:"num"`
	Timeout string `json:"timeout"`
}

// RoutingInfo describes the current routing configuration.
type RoutingInfo struct {
	Mode     string   `json:"mode"`
	Pool     []string `json:"pool"`
	Fallback []string `json:"fallback"`
}

// AdapterInfo describes a single adapter entry.
type AdapterInfo struct {
	Bin       string            `json:"bin"`
	Available bool              `json:"available"`
	Timeout   string            `json:"timeout,omitempty"`
	Num       int               `json:"num,omitempty"`
	// Config shows the raw (pre-expansion) template strings from the config file
	// so agents know which env vars are required — resolved secrets are never shown.
	Config    map[string]string `json:"config,omitempty"`
}

// Build assembles an AgentInfo from a resolved Config.
// configSource is the path of the config file that was loaded (may be empty
// if no file was found). rawAdapterConfigs carries the unexpanded config maps
// (i.e. still containing ${VAR} placeholders) so secrets are never emitted.
func Build(
	version string,
	cfg *config.Config,
	configSource string,
	rawAdapterConfigs map[string]map[string]string,
) AgentInfo {
	ai := AgentInfo{
		Tool: ToolInfo{
			Name:    "qry",
			Version: version,
			Description: "A terminal-native, agent-first web search hub. " +
				"Routes queries through pluggable adapter binaries and always outputs JSON.",
			Usage: "qry [flags] <query>",
			Flags: map[string]Flag{
				"--adapter": {
					Description: "Use a specific adapter by name, bypassing routing config.",
					Default:     "",
				},
				"--mode": {
					Description: "Routing mode: \"first\" (sequential, stop on success) or \"merge\" (concurrent, combined results). Overrides config.",
					Default:     "",
				},
				"--num": {
					Description: "Number of results to return. Overrides config.",
					Default:     "10",
				},
				"--timeout": {
					Description: "Per-adapter timeout, e.g. \"5s\". Overrides config.",
					Default:     "5s",
				},
				"--config": {
					Description: "Path to an alternate config file.",
					Default:     "~/.config/qry/config.toml",
				},
				"--site": {
					Description: "Restrict results to a domain. Repeatable. Prepends \"site:<domain>\" (joined with OR) to the query. Example: --site docs.rs --site crates.io",
					Default:     "",
				},
				"--agent-info": {
					Description: "Print this agent-info JSON and exit. No query required.",
					Default:     "",
				},
			},
		},
		RoutingModes: map[string]string{
			"first": "Tries pool adapters in order; returns on first success. " +
				"Falls back to the fallback list if all pool adapters fail. Fast, good for most use cases.",
			"merge": "Queries all pool adapters concurrently, deduplicates results by URL, " +
				"and returns a combined set. Partial failure is non-fatal — results from " +
				"successful adapters are returned alongside warnings.",
		},
	}

	if cfg != nil {
		ai.Config = buildConfigInfo(cfg, configSource, rawAdapterConfigs)
	}

	return ai
}

func buildConfigInfo(
	cfg *config.Config,
	source string,
	rawAdapterConfigs map[string]map[string]string,
) *ConfigInfo {
	ci := &ConfigInfo{
		Source: source,
		Defaults: DefaultsInfo{
			Num:     cfg.Defaults.Num,
			Timeout: formatDuration(cfg.Defaults.Timeout),
		},
		Routing: RoutingInfo{
			Mode:     cfg.Routing.Mode,
			Pool:     cfg.Routing.Pool,
			Fallback: cfg.Routing.Fallback,
		},
		Adapters: make(map[string]AdapterInfo, len(cfg.Adapters)),
	}

	for name, adapter := range cfg.Adapters {
		_, err := os.Stat(adapter.Bin)
		adapterInfo := AdapterInfo{
			Bin:       adapter.Bin,
			Available: err == nil,
		}
		if adapter.Timeout > 0 {
			adapterInfo.Timeout = formatDuration(adapter.Timeout)
		}
		if adapter.Num > 0 {
			adapterInfo.Num = adapter.Num
		}
		// Use raw (unexpanded) config so ${VAR} templates are visible, not secrets.
		if raw, ok := rawAdapterConfigs[name]; ok && len(raw) > 0 {
			adapterInfo.Config = raw
		}
		ci.Adapters[name] = adapterInfo
	}

	return ci
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}
