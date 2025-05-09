package cilium

import (
	"github.com/Azure/mcp-kubernetes/go/internal/command"
	"github.com/Azure/mcp-kubernetes/go/internal/config"
	"github.com/Azure/mcp-kubernetes/go/internal/security"
)

// CiliumExecutor implements the CommandExecutor interface for cilium commands
type CiliumExecutor struct{}

// NewExecutor creates a new CiliumExecutor instance
func NewExecutor() *CiliumExecutor {
	return &CiliumExecutor{}
}

// Execute handles cilium command execution
func (e *CiliumExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
	ciliumCmd := params["command"].(string)

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(ciliumCmd, security.CommandTypeCilium)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	// Execute the command
	process := command.NewShellProcess("cilium", cfg.Timeout)
	output, err := process.Run(ciliumCmd)
	if err != nil {
		return map[string]interface{}{
			"error": "Command execution error: " + err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"text": output,
	}, nil
}
