# -*- coding: utf-8 -*-
import re
import subprocess
from typing import List, Union, Optional

# Define read-only operations
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


class ShellProcess:
    """Wrapper for shell command."""

    def __init__(
        self,
        command: str = "/bin/bash",
        strip_newlines: bool = False,
        return_err_output: bool = True,
    ):
        """Initialize with stripping newlines."""
        self.strip_newlines = strip_newlines
        self.return_err_output = return_err_output
        self.command = command

    def run(self, args: Union[str, List[str]], input=None) -> str:
        """Run the command."""
        if isinstance(args, str):
            args = [args]
        commands = ";".join(args)
        if not commands.startswith(self.command):
            commands = f"{self.command} {commands}"

        return self.exec(commands, input=input)

    def exec(self, commands: Union[str, List[str]], input=None) -> str:
        """Run commands and return final output."""
        if isinstance(commands, str):
            commands = [commands]
        commands = ";".join(commands)
        try:
            output = subprocess.run(
                commands,
                shell=True,
                check=True,
                input=input,
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
            ).stdout.decode()
        except subprocess.CalledProcessError as error:
            if self.return_err_output:
                return error.stdout.decode()
            return str(error)
        if self.strip_newlines:
            output = output.strip()
        return output


def is_read_operation(command: str, allowed_operations: List[str]) -> bool:
    """Check if a command is a read operation."""
    # Split command by spaces and get the first non-option argument
    cmd_parts = command.split()
    operation = None

    for part in cmd_parts:
        if not part.startswith("-"):
            if part != "kubectl" and part != "helm":
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


async def kubectl(command: str) -> str:
    """Run a kubectl command and return the output."""
    # Import here to avoid circular import
    from .security import security_config

    # Check if this is a write operation in read-only mode
    if security_config.readonly and not is_read_operation(
        command, KUBECTL_READ_OPERATIONS
    ):
        return "Error: Cannot execute write operations in read-only mode"

    # Extract namespace from command
    namespace = extract_namespace_from_command(command)

    # If command applies to all namespaces (--all-namespaces or -A), and there are namespace restrictions
    if namespace == "*" and (
        security_config.allowed_namespaces or security_config.denied_namespaces
    ):
        return "Error: Access to all namespaces is restricted by security configuration"

    # If a namespace is specified (or default "default" is used), check if it's allowed
    if namespace and namespace != "*":
        if not security_config.is_namespace_allowed(namespace):
            return f"Error: Access to namespace '{namespace}' is denied by security configuration"

    process = ShellProcess(command="kubectl")
    output = process.run(command)
    return output


async def helm(command: str) -> str:
    """Run a helm command and return the output."""
    # Import here to avoid circular import
    from .security import security_config

    # Check if this is a write operation in read-only mode
    if security_config.readonly and not is_read_operation(
        command, HELM_READ_OPERATIONS
    ):
        return "Error: Cannot execute write operations in read-only mode"

    # Extract namespace from command
    namespace = extract_namespace_from_command(command)

    # If a namespace is specified, check if it's allowed
    if namespace and namespace != "*":
        if not security_config.is_namespace_allowed(namespace):
            return f"Error: Access to namespace '{namespace}' is denied by security configuration"

    process = ShellProcess(command="helm")
    output = process.run(command)
    return output
