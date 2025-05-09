package helm

import (
	"fmt"

	"github.com/Azure/mcp-kubernetes/go/internal/command"
	"github.com/Azure/mcp-kubernetes/go/internal/config"
	"github.com/Azure/mcp-kubernetes/go/internal/security"
	"github.com/Azure/mcp-kubernetes/go/internal/tools"
)

// HelmExecutor implements the CommandExecutor interface for helm commands
type HelmExecutor struct{}

var _ tools.CommandExecutor = (*HelmExecutor)(nil)

// NewExecutor creates a new HelmExecutor instance
func NewExecutor() *HelmExecutor {
	return &HelmExecutor{}
}

// Execute handles helm command execution
func (e *HelmExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	helmCmd, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("invalid command parameter")
	}

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(helmCmd, security.CommandTypeHelm)
	if err != nil {
		return "", err
	}

	// Execute the command
	process := command.NewShellProcess("helm", cfg.Timeout)
	return process.Run(helmCmd)
}
