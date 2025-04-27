class SecurityConfig:
    """Security configuration for MCP Kubernetes Server."""

    def __init__(self):
        self._readonly: bool = False
        self._allowed_namespaces = (
            []
        )  # List of allowed namespaces, empty list means no restrictions
        self._denied_namespaces = (
            []
        )  # List of denied namespaces, takes precedence over allowed_namespaces

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
        return self._allowed_namespaces

    @allowed_namespaces.setter
    def allowed_namespaces(self, namespaces: str):
        """Set the list of allowed namespaces."""
        self._allowed_namespaces = [ns.strip() for ns in namespaces.split(",")]

    @property
    def denied_namespaces(self):
        """Get the list of denied namespaces."""
        return self._denied_namespaces

    @denied_namespaces.setter
    def denied_namespaces(self, namespaces: str):
        """Set the list of denied namespaces."""
        self._denied_namespaces = [ns.strip() for ns in namespaces.split(",")]

    def is_namespace_allowed(self, namespace):
        """Check if a namespace is allowed to be accessed"""
        # TODO: shall we support regex for namespace?

        # If namespace is in the denied list, reject it directly
        if namespace in self._denied_namespaces:
            return False

        # If the allowed list is empty, allow all namespaces (except those in the denied list)
        if not self._allowed_namespaces:
            return True

        # Otherwise, only allow namespaces in the allowed list
        return namespace in self._allowed_namespaces


security_config = SecurityConfig()
