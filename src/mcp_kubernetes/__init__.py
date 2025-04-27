from .main import server

def main():
    """Main entry point for the mcp_kubernetes module."""
    server()

__all__ = [
    "main",
    "server",
]
