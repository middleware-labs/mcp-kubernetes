package kubectl

import (
	"strings"

	"github.com/Azure/mcp-kubernetes/go/internal/command"
	"github.com/Azure/mcp-kubernetes/go/internal/config"
	"github.com/Azure/mcp-kubernetes/go/internal/security"
	"github.com/Azure/mcp-kubernetes/go/internal/tools"
)

// KubectlExecutor implements the CommandExecutor interface for kubectl commands
type KubectlExecutor struct{}

// This line ensures KubectlExecutor implements the CommandExecutor interface
var _ tools.CommandExecutor = (*KubectlExecutor)(nil)

// NewExecutor creates a new KubectlExecutor instance
func NewExecutor() *KubectlExecutor {
	return &KubectlExecutor{}
}

// Execute handles general kubectl command execution (for backward compatibility)
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
	return e.executeCommand(kubectlCmd, "", cfg)
}

// ExecuteSpecificCommand executes a specific kubectl command with the given arguments
func (e *KubectlExecutor) ExecuteSpecificCommand(cmd string, params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
	args, ok := params["args"].(string)
	if !ok {
		args = ""
	}

	// Build the full kubectl command for validation
	fullCmd := cmd
	if args != "" {
		fullCmd += " " + args
	}

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(fullCmd, security.CommandTypeKubectl)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}, nil
	}

	// Execute the command
	return e.executeCommand(cmd, args, cfg)
}

// executeCommand executes a kubectl command with the given arguments
// and returns the formatted result
func (e *KubectlExecutor) executeCommand(cmd string, args string, cfg *config.ConfigData) (interface{}, error) {
	output, err := e.executeKubectlCommand(cmd, args, cfg)
	if err != nil {
		return map[string]interface{}{
			"error": "Command execution error: " + err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"text": output,
	}, nil
}

// CreateCommandExecutor creates a CommandExecutor for a specific kubectl command
func CreateCommandExecutor(cmd string) func(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
	return func(params map[string]interface{}, cfg *config.ConfigData) (interface{}, error) {
		executor := NewExecutor()
		return executor.ExecuteSpecificCommand(cmd, params, cfg)
	}
}

// executeKubectlCommand executes a kubectl command with the given arguments
func (e *KubectlExecutor) executeKubectlCommand(cmd string, args string, cfg *config.ConfigData) (string, error) {
	process := command.NewShellProcess("kubectl", cfg.Timeout)

	var fullCmd string
	if strings.HasPrefix(cmd, "kubectl ") {
		// If command already includes "kubectl", use it as is (for backward compatibility)
		fullCmd = cmd
	} else {
		// Otherwise build the command
		fullCmd = "kubectl " + cmd
		if args != "" {
			fullCmd += " " + args
		}
	}

	return process.Run(fullCmd)
}
