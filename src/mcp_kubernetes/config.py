from mcp_kubernetes.security import SecurityConfig


class Config:
    def __init__(self):
        self.disable_helm = False
        self.timeout: int = 60
        self.security_config = SecurityConfig()


config = Config()
