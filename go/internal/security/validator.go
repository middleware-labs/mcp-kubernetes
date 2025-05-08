package security

import (
	"regexp"
	"strings"
)

// Command type constants
const (
	CommandTypeKubectl = "kubectl"
	CommandTypeHelm    = "helm"
	CommandTypeCilium  = "cilium"
)

var (
	// KubectlReadOperations defines kubectl operations that don't modify state
	KubectlReadOperations = []string{
		"get", "describe", "explain", "logs", "top", "auth", "config",
		"cluster-info", "api-resources", "api-versions", "version", "diff",
		"completion", "help", "kustomize", "options", "plugin", "proxy", "wait", "cp",
	}

	// HelmReadOperations defines helm operations that don't modify state
	HelmReadOperations = []string{
		"get", "history", "list", "show", "status", "search", "repo",
		"env", "version", "verify", "completion", "help",
	}

	// CiliumReadOperations defines cilium operations that don't modify state
	CiliumReadOperations = []string{
		"status", "version", "config", "help", "context", "connectivity",
		"endpoint", "identity", "ip", "map", "metrics", "monitor", "policy",
		"hubble", "bpf", "list", "observe", "service",
	}
)

// Validator handles validation of commands against security configuration
type Validator struct {
	secConfig *SecurityConfig
}

// NewValidator creates a new Validator instance with the given security configuration
func NewValidator(secConfig *SecurityConfig) *Validator {
	return &Validator{
		secConfig: secConfig,
	}
}

// ValidationError represents a security validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// getReadOperationsList returns the appropriate list of read operations based on command type
func (v *Validator) getReadOperationsList(commandType string) []string {
	switch commandType {
	case CommandTypeKubectl:
		return KubectlReadOperations
	case CommandTypeHelm:
		return HelmReadOperations
	case CommandTypeCilium:
		return CiliumReadOperations
	default:
		return []string{}
	}
}

// ValidateCommand validates a command against all security settings
func (v *Validator) ValidateCommand(command, commandType string) error {
	readOperations := v.getReadOperationsList(commandType)
	
	// Check readonly restrictions
	if err := v.validateReadOnly(command, readOperations); err != nil {
		return err
	}

	// Check namespace scope restrictions
	if err := v.validateNamespaceScope(command); err != nil {
		return err
	}

	return nil
}

// validateReadOnly validates if a command is allowed in read-only mode
func (v *Validator) validateReadOnly(command string, readOperations []string) error {
	// Check if we're in readonly mode and if this is a write operation
	if v.secConfig.ReadOnly && !v.isReadOperation(command, readOperations) {
		return &ValidationError{Message: "Error: Cannot execute write operations in read-only mode"}
	}

	return nil
}

// validateNamespaceScope validates if a command's namespace scope is allowed by security settings
func (v *Validator) validateNamespaceScope(command string) error {
	// Extract namespace from command
	namespace := v.extractNamespaceFromCommand(command)

	// If command applies to all namespaces, and there are namespace restrictions
	if namespace == "*" && (len(v.secConfig.allowedNamespaces) > 0 || len(v.secConfig.allowedNamespacesRe) > 0) {
		return &ValidationError{Message: "Error: Access to all namespaces is restricted by security configuration"}
	}

	// If a namespace is specified (or default "default" is used), check if it's allowed
	if namespace != "" && namespace != "*" {
		if !v.secConfig.IsNamespaceAllowed(namespace) {
			return &ValidationError{
				Message: "Error: Access to namespace '" + namespace + "' is denied by security configuration",
			}
		}
	}

	return nil
}

// isReadOperation checks if a command is a read operation
func (v *Validator) isReadOperation(command string, allowedOperations []string) bool {
	cmdParts := strings.Fields(command)
	var operation string

	for _, part := range cmdParts {
		if !strings.HasPrefix(part, "-") {
			// Skip the initial command name (kubectl, helm, cilium)
			if part != CommandTypeKubectl && part != CommandTypeHelm && part != CommandTypeCilium {
				operation = part
				break
			}
		}
	}

	for _, allowed := range allowedOperations {
		if operation == allowed {
			return true
		}
	}

	return false
}

// extractNamespaceFromCommand extracts the namespace from a command
func (v *Validator) extractNamespaceFromCommand(command string) string {
	// Check for explicit namespace parameter
	namespacePattern := `(?:-n|--namespace)[\s=]([^\s]+)`
	re := regexp.MustCompile(namespacePattern)
	matches := re.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}

	// Check if there's a format like <resource>/<name>
	resourcePattern := `(\S+)/(\S+)`
	re = regexp.MustCompile(resourcePattern)
	if re.MatchString(command) {
		// If the command contains resource/name format but no explicit namespace,
		// the default namespace "default" will be used
		return "default"
	}

	// Check for --all-namespaces or -A
	if strings.Contains(command, "--all-namespaces") || strings.Contains(command, " -A") || strings.HasSuffix(command, " -A") {
		return "*" // Special marker indicating all namespaces
	}

	return "" // No namespace found, default namespace will be used
}
