"""
Tool registry module for MCP Kubernetes.

This module provides functionality to register and categorize kubectl functions
with specific labels, allowing for filtering and selection based on those labels.
"""

# Constants for function type labels
KUBECTL_READONLY = "readonly"  # Read-only operations
KUBECTL_RW = "rw"  # Read-write operations
KUBECTL_ADMIN = "admin"  # Admin operations

# Global registry for function labels and objects
KUBECTL_FUNCTION_REGISTRY = {}


def kubectl_func_register(label: str):
    """
    Decorator to label kubectl functions with a specific category.

    Args:
        label (str): The label to assign to the function (e.g., "readonly", "rw", "admin").

    Returns:
        function: The wrapped function with the label applied.

    Example:
        @kubectl_func_register("readonly")
        def kubectl_get(command: str) -> str:
            # Function implementation
            pass
    """

    def decorator(func):
        # Store the function object with its label
        KUBECTL_FUNCTION_REGISTRY[func.__name__] = {"function": func, "label": label}
        return func

    return decorator


def get_kubectl_functions_by_label(label: str) -> list[callable]:
    """
    Retrieve all kubectl functions with a specific label.

    Args:
        label (str): The label to filter by (e.g., "readonly", "rw", "admin").

    Returns:
        list: A list of function objects with the specified label.

    Example:
        readonly_functions = get_kubectl_functions_by_label("readonly")
        # Returns a list of function objects that can be directly called
    """
    return [
        entry["function"]
        for name, entry in KUBECTL_FUNCTION_REGISTRY.items()
        if entry["label"] == label
    ]


def get_all_kubectl_registered_functions() -> list[callable]:
    """
    Get all registered kubectl functions.

    Returns:
        list: A list of all registered function objects.
    """
    return [entry["function"] for entry in KUBECTL_FUNCTION_REGISTRY.values()]
