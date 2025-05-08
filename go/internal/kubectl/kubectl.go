package kubectl

import (
	"github.com/Azure/mcp-kubernetes/go/internal/command"
	"github.com/Azure/mcp-kubernetes/go/internal/config"
	"github.com/Azure/mcp-kubernetes/go/internal/security"
)

// KubectlExecutor implements the CommandExecutor interface for kubectl commands
type KubectlExecutor struct{}

// NewExecutor creates a new KubectlExecutor instance
func NewExecutor() *KubectlExecutor {
	return &KubectlExecutor{}
}

// Execute handles kubectl command execution
func (e *KubectlExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
	kubectlCmd := params["command"].(string)

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(kubectlCmd, security.CommandTypeKubectl)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	// Execute the command
	process := command.NewShellProcess("kubectl", cfg.Timeout)
	output, err := process.Run(kubectlCmd)
	if err != nil {
		return map[string]interface{}{
			"error": "Command execution error: " + err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"text": output,
	}, nil
}

// ExecuteKubectl handles kubectl command execution (legacy function for backward compatibility)
func ExecuteKubectl(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
	executor := NewExecutor()
	return executor.Execute(params, cfg)
}
