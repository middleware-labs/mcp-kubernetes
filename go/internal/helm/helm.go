package helm

import (
	"github.com/Azure/mcp-kubernetes/go/internal/command"
	"github.com/Azure/mcp-kubernetes/go/internal/config"
	"github.com/Azure/mcp-kubernetes/go/internal/security"
)

// HelmExecutor implements the CommandExecutor interface for helm commands
type HelmExecutor struct{}

// NewExecutor creates a new HelmExecutor instance
func NewExecutor() *HelmExecutor {
	return &HelmExecutor{}
}

// Execute handles helm command execution
func (e *HelmExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
	helmCmd := params["command"].(string)

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(helmCmd, security.CommandTypeHelm)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	// Execute the command
	process := command.NewShellProcess("helm", cfg.Timeout)
	output, err := process.Run(helmCmd)
	if err != nil {
		return map[string]interface{}{
			"error": "Command execution error: " + err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"text": output,
	}, nil
}

// ExecuteHelm handles helm command execution (legacy function for backward compatibility)
func ExecuteHelm(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
	executor := NewExecutor()
	return executor.Execute(params, cfg)
}
