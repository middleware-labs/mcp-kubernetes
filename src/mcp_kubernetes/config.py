from mcp_kubernetes.security import SecurityConfig
from typing import Set


class Config:
    def __init__(self):
        # Define all available tools - kubectl is mandatory and not included here
        self.available_tools = {"kubectl", "helm", "cilium"}
        # Set of enabled tools (will be populated from command line args)
        self.additional_tools: Set[str] = set()
        self.timeout: int = 60
        self.security_config = SecurityConfig()


config = Config()
