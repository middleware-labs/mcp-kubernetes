package kubectl

import (
	"crypto/sha1"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Azure/mcp-kubernetes/pkg/command"
	"github.com/Azure/mcp-kubernetes/pkg/config"
	"github.com/Azure/mcp-kubernetes/pkg/security"
	"github.com/Azure/mcp-kubernetes/pkg/tools"
)

type KubectlExecutor struct {
	pulsarWorker *Worker
}

var _ tools.CommandExecutor = (*KubectlExecutor)(nil)

func NewExecutor(pulsarWorker *Worker) *KubectlExecutor {
	return &KubectlExecutor{pulsarWorker: pulsarWorker}
}

func (e *KubectlExecutor) executeKubectlCommand(cmd string, args string, cfg *config.ConfigData) (string, error) {
	process := command.NewShellProcess("kubectl", cfg.Timeout)

	var fullCmd string
	if strings.HasPrefix(cmd, "kubectl ") {
		fullCmd = cmd
	} else {
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
		fullCmd = cmd
	} else {
		fullCmd = "kubectl " + cmd
		if args != "" {
			fullCmd += " " + args
		}
	}

	id := int(time.Now().UnixMilli())
	respCh := make(chan string, 1)
	e.pulsarWorker.pending.Store(id, respCh)

	topic := fmt.Sprintf("mcp-%s-%x",
		strings.ToLower(e.pulsarWorker.cfg.Token),
		sha1.Sum([]byte(strings.ToLower(e.pulsarWorker.cfg.Location))))

	// LOG: full request context. Compare send_topic vs worker_topic immediately.
	slog.Info("executeKubectlCommandOnHost start",
		"id", id,
		"full_cmd", fullCmd,
		"send_topic", topic,
		"worker_topic", e.pulsarWorker.topic,
		"topics_match", topic == e.pulsarWorker.topic,
		"location", e.pulsarWorker.cfg.Location,
		"location_lower", strings.ToLower(e.pulsarWorker.cfg.Location),
		"token_prefix", safePrefix(e.pulsarWorker.cfg.Token, 8),
		"timeout_seconds", e.pulsarWorker.cfg.Timeout)

	err := e.pulsarWorker.sendRequest(e.pulsarWorker.cfg.AccountUID, id, topic, map[string]interface{}{
		"command": fullCmd,
	})
	if err != nil {
		e.pulsarWorker.pending.Delete(id)
		slog.Error("sendRequest failed",
			"id", id, "topic", topic, "error", err.Error())
		return "", fmt.Errorf("failed to send request: %s", err.Error())
	}

	slog.Info("waiting for response",
		"id", id,
		"topic", topic,
		"timeout_seconds", e.pulsarWorker.cfg.Timeout,
		"pending_count_before_wait", len(e.pulsarWorker.snapshotPendingIDs()))

	waitStart := time.Now()

	select {
	case res := <-respCh:
		// LOG: got it. Size + how long it took.
		slog.Info("response delivered",
			"id", id,
			"topic", topic,
			"elapsed_ms", time.Since(waitStart).Milliseconds(),
			"response_size", len(res),
			"response_preview", safePrefix(res, 300))
		return res, nil

	case <-time.After(time.Second * time.Duration(e.pulsarWorker.cfg.Timeout)):
		e.pulsarWorker.pending.Delete(id)
		// LOG: timeout. Dump every clue you have.
		slog.Error("TIMEOUT waiting for response",
			"id", id,
			"send_topic", topic,
			"worker_topic", e.pulsarWorker.topic,
			"topics_match", topic == e.pulsarWorker.topic,
			"elapsed_ms", time.Since(waitStart).Milliseconds(),
			"timeout_seconds", e.pulsarWorker.cfg.Timeout,
			"still_pending_ids", e.pulsarWorker.snapshotPendingIDs(),
			"full_cmd", fullCmd)
		return "", fmt.Errorf("timeout waiting for response")
	}
}

func (e *KubectlExecutor) Execute(params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	kubectlCmd, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("invalid command parameter")
	}

	slog.Info("Execute called", "command", kubectlCmd, "param_keys", mapKeys(params))

	validator := security.NewValidator(cfg.SecurityConfig)
	if err := validator.ValidateCommand(kubectlCmd, security.CommandTypeKubectl); err != nil {
		slog.Error("Execute validation failed", "command", kubectlCmd, "error", err)
		return "", err
	}

	return e.executeKubectlCommand(kubectlCmd, "", cfg)
}

func (e *KubectlExecutor) ExecuteSpecificCommand(cmd string, params map[string]interface{}, cfg *config.ConfigData) (string, error) {
	args, ok := params["args"].(string)
	if !ok {
		args = ""
	}

	fullCmd := cmd
	if args != "" {
		fullCmd += " " + args
	}

	slog.Info("ExecuteSpecificCommand called",
		"cmd", cmd,
		"args", args,
		"full_cmd", fullCmd,
		"param_keys", mapKeys(params))

	validator := security.NewValidator(cfg.SecurityConfig)
	if err := validator.ValidateCommand(fullCmd, security.CommandTypeKubectl); err != nil {
		slog.Error("ExecuteSpecificCommand validation failed", "full_cmd", fullCmd, "error", err)
		return "", err
	}

	return e.executeKubectlCommand(cmd, args, cfg)
}
