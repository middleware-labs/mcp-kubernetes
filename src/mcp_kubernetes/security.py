import logging
import re

logger = logging.getLogger(__name__)


class SecurityConfig:
    """Security configuration for MCP Kubernetes Server."""

    # Special characters that indicate a pattern is a regex
    REGEX_SPECIAL_CHARS = set(".*+?[](){}|^$\\")

    def __init__(self):
        self._readonly: bool = False
        self._allowed_namespaces = (
            []
        )  # List of allowed namespaces, empty list means no restrictions
        self._allowed_namespaces_re = (
            []
        )  # List of compiled regex patterns for namespace matching

    @property
    def readonly(self):
        """Check read-only mode."""
        return self._readonly

    @readonly.setter
    def readonly(self, value: bool):
        """Set read-only mode."""
        self._readonly = value

    @property
    def allowed_namespaces(self):
        """Get the list of allowed namespaces."""
        return self._allowed_namespaces + self._allowed_namespaces_re

    @staticmethod
    def _is_regex_pattern(pattern: str) -> bool:
        """
        Determine if a pattern is likely a regex pattern based on special characters.

        Args:
            pattern: The pattern to check

        Returns:
            True if the pattern contains regex special characters, False otherwise
        """
        return any(c in SecurityConfig.REGEX_SPECIAL_CHARS for c in pattern)

    @allowed_namespaces.setter
    def allowed_namespaces(self, namespaces: str):
        """
        Set the list of allowed namespaces.

        Namespace patterns can be:
        1. Literal namespace names (e.g., "default", "kube-system")
        2. Regex patterns containing special characters (e.g., "app-.*", "test-\\d+")

        Patterns with regex special characters are automatically treated as regex patterns.
        """
        self._allowed_namespaces = []
        self._allowed_namespaces_re = []

        if not namespaces:
            return

        for ns in namespaces.split(","):
            ns = ns.strip()
            if not ns:
                continue

            if self._is_regex_pattern(ns):
                try:
                    self._allowed_namespaces_re.append(re.compile(f"^{ns}$"))
                except re.error:
                    # If regex is invalid, treat it as a literal string
                    logger.warning(
                        f"Invalid regex pattern '{ns}' provided. Treating as literal string."
                    )
                    self._allowed_namespaces.append(ns)
            else:
                self._allowed_namespaces.append(ns)

    def is_namespace_allowed(self, namespace):
        """
        Check if a namespace is allowed to be accessed.

        A namespace is allowed if:
        1. No restrictions are defined (empty allowed_namespaces and empty allowed_namespaces_re)
        2. The namespace exactly matches one in the allowed list
        3. The namespace matches a regex pattern in the allowed list
        """
        # If no restrictions are defined, allow all namespaces
        if not self._allowed_namespaces and not self._allowed_namespaces_re:
            return True

        # Check for exact match in allowed namespaces
        if namespace in self._allowed_namespaces:
            return True

        # Check for match against regex patterns
        for pattern in self._allowed_namespaces_re:
            if pattern.match(namespace):
                return True

        return False
