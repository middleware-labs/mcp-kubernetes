package config

import (
	"strings"

	"github.com/Azure/mcp-kubernetes/internal/security"
	flag "github.com/spf13/pflag"
)

// ConfigData holds the global configuration
type ConfigData struct {
	// Map of additional tools enabled
	AdditionalTools map[string]bool
	// Command execution timeout in seconds
	Timeout int
	// Security configuration
	SecurityConfig *security.SecurityConfig

	// Command-line specific options
	Transport       string
	Port            int
	ReadOnly        bool
	AllowNamespaces string
}

// NewConfig creates and returns a new configuration instance
func NewConfig() *ConfigData {
	return &ConfigData{
		AdditionalTools: make(map[string]bool),
		Timeout:         60,
		SecurityConfig:  security.NewSecurityConfig(),
		Transport:       "stdio",
		Port:            8000,
		ReadOnly:        false,
		AllowNamespaces: "",
	}
}

// ParseFlags parses command line arguments and updates the configuration
func (cfg *ConfigData) ParseFlags() {
	// Server configuration
	flag.StringVar(&cfg.Transport, "transport", "stdio", "Transport mechanism to use (stdio or sse)")
	flag.IntVar(&cfg.Port, "port", 8000, "Port to use for the server (only used with sse transport)")
	flag.IntVar(&cfg.Timeout, "timeout", 60, "Timeout for command execution in seconds, default is 60s")

	// Tools configuration
	additionalTools := flag.String("additional-tools", "",
		"Comma-separated list of additional tools to support (kubectl is always enabled). Available: helm,cilium")

	// Security settings
	flag.BoolVar(&cfg.ReadOnly, "readonly", false, "Enable read-only mode (prevents write operations)")
	flag.StringVar(&cfg.AllowNamespaces, "allow-namespaces", "",
		"Comma-separated list of namespaces to allow (empty means all allowed)")

	flag.Parse()

	// Update security config
	cfg.SecurityConfig.ReadOnly = cfg.ReadOnly
	if cfg.AllowNamespaces != "" {
		cfg.SecurityConfig.SetAllowedNamespaces(cfg.AllowNamespaces)
	}

	// Parse additional tools
	if *additionalTools != "" {
		for _, tool := range strings.Split(*additionalTools, ",") {
			tool = strings.TrimSpace(tool)
			if tool == "" {
				continue
			}
			cfg.AdditionalTools[tool] = true
		}
	}
}

var availableTools = []string{"kubectl", "helm", "cilium"}

// IsToolSupported checks if a tool is supported
func IsToolSupported(tool string) bool {
	for _, t := range availableTools {
		if t == tool {
			return true
		}
	}
	return false
}

// AvailableTools returns the list of available tools
func AvailableTools() []string {
	return availableTools
}
