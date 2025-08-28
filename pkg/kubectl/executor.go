package kubectl

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/mcp-kubernetes/pkg/command"
	"github.com/Azure/mcp-kubernetes/pkg/config"
	"github.com/Azure/mcp-kubernetes/pkg/security"
	"github.com/Azure/mcp-kubernetes/pkg/tools"
)

// KubectlExecutor implements the CommandExecutor interface for kubectl commands
type KubectlExecutor struct {
	pulsarWorker *Worker // Pulsar worker for command execution
}

// This line ensures KubectlExecutor implements the CommandExecutor interface
var _ tools.CommandExecutor = (*KubectlExecutor)(nil)

// NewExecutor creates a new KubectlExecutor instance
func NewExecutor(pulsarWorker *Worker) *KubectlExecutor {
	return &KubectlExecutor{
		pulsarWorker: pulsarWorker,
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

func (e *KubectlExecutor) executeKubectlCommandOnHost(cmd string, args string, cfg *config.ConfigData) (string, error) {
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
	id := int(time.Now().UnixMilli())
	topic := fmt.Sprintf("agent-%s-%x", strings.ToLower(e.pulsarWorker.cfg.Token), sha1.Sum([]byte(strings.ToLower(e.pulsarWorker.cfg.Location))))
	err := e.pulsarWorker.sendRequest(e.pulsarWorker.cfg.AccountUID, id, topic, map[string]interface{}{
		"command": fullCmd,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send request: %s", err.Error())
	}
	return e.pulsarWorker.SubscribeUpdates(topic+"-unsubscribe", e.pulsarWorker.cfg.Token, id, e.pulsarWorker.cfg.Timeout)
}

// Validate the command against security settings}

// Execute handles general kubectl command execution (for backward compatibility)
func (e *KubectlExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	kubectlCmd, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("invalid command parameter")
	}

	// Validate the command against security settings
	validator := security.NewValidator(cfg.SecurityConfig)
	err := validator.ValidateCommand(kubectlCmd, security.CommandTypeKubectl)
	if err != nil {
		return "", err
	}

	// Execute the command
	return e.executeKubectlCommand(kubectlCmd, "", cfg)
}

// ExecuteSpecificCommand executes a specific kubectl command with the given arguments
func (e *KubectlExecutor) ExecuteSpecificCommand(cmd string, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
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
		return "", err
	}

	// Execute the command
	return e.executeKubectlCommand(cmd, args, cfg)
}
