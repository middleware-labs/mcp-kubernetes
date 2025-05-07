# Define read-only operations
import re
from typing import List, Optional


KUBECTL_READ_OPERATIONS = [
    "get",
    "describe",
    "explain",
    "logs",
    "top",
    "auth",
    "config",
    "cluster-info",
    "api-resources",
    "api-versions",
    "version",
    "diff",
    "completion",
    "help",
    "kustomize",
    "options",
    "plugin",
    "proxy",
    "wait",
    "cp",
]

HELM_READ_OPERATIONS = [
    "get",
    "history",
    "list",
    "show",
    "status",
    "search",
    "repo",
    "env",
    "version",
    "verify",
    "completion",
    "help",
]

CILIUM_READ_OPERATIONS = [
    "status",
    "version",
    "config",
    "help",
    "context",
    "connectivity",
    "endpoint",
    "identity",
    "ip",
    "map",
    "metrics",
    "monitor",
    "policy",
    "hubble",
    "bpf",
    "list",
    "observe",
    "service",
]


def is_read_operation(command: str, allowed_operations: List[str]) -> bool:
    """Check if a command is a read operation."""
    # Split command by spaces and get the first non-option argument
    cmd_parts = command.split()
    operation = None

    for part in cmd_parts:
        if not part.startswith("-"):
            if part != "kubectl" and part != "helm" and part != "cilium":
                operation = part
                break

    return operation in allowed_operations


def extract_namespace_from_command(command: str) -> Optional[str]:
    """
    Extract namespace from command.

    Check for -n/--namespace parameter or parse specific resource path.
    If no namespace is specified, return None (indicating default namespace).
    """
    # First check if there's an explicit namespace parameter
    namespace_pattern = r"(?:-n|--namespace)[\s=]([^\s]+)"
    match = re.search(namespace_pattern, command)
    if match:
        return match.group(1)

    # Check if there's a format like <resource>/<name> -n <namespace>
    resource_pattern = r"(\S+)/(\S+)"
    if re.search(resource_pattern, command):
        # If the command contains resource/name format but no explicit namespace,
        # the default namespace "default" will be used
        return "default"

    # If command contains --all-namespaces or -A, it applies to all namespaces
    if "--all-namespaces" in command or "-A" in command:
        return "*"  # Special marker indicating all namespaces

    return None  # No namespace found, default namespace will be used


def validate_readonly(command: str, read_operations: List[str]) -> Optional[str]:
    """
    Validate if a command is allowed in read-only mode.

    Returns an error message string if validation fails, None if validation passes.
    """
    # Import here to avoid circular import
    from .config import config

    # Check if we're in readonly mode and if this is a write operation
    if config.security_config.readonly and not is_read_operation(
        command, read_operations
    ):
        return "Error: Cannot execute write operations in read-only mode"

    return None


def validate_namespace_scope(command: str) -> Optional[str]:
    """
    Validate if a command's namespace scope is allowed by security settings.

    Returns an error message string if validation fails, None if validation passes.
    """
    # Import here to avoid circular import
    from .config import config

    # Extract namespace from command
    namespace = extract_namespace_from_command(command)

    # If command applies to all namespaces (--all-namespaces or -A), and there are namespace restrictions
    if namespace == "*" and config.security_config.allowed_namespaces:
        return "Error: Access to all namespaces is restricted by security configuration"

    # If a namespace is specified (or default "default" is used), check if it's allowed
    if namespace and namespace != "*":
        if not config.security_config.is_namespace_allowed(namespace):
            return f"Error: Access to namespace '{namespace}' is denied by security configuration"

    return None


def validate_command(
    command: str, read_operations: List[str], command_type: str
) -> Optional[str]:
    """
    Validate a command against all security settings.

    Returns an error message string if any validation fails, None if all validations pass.
    """
    # Check readonly restrictions
    error = validate_readonly(command, read_operations)
    if error:
        return error

    # Check namespace scope restrictions
    error = validate_namespace_scope(command)
    if error:
        return error

    return None
