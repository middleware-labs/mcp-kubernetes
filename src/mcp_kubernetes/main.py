# -*- coding: utf-8 -*-
import argparse
from fastmcp import FastMCP
import logging
from .kubeclient import setup_client, apis, crds, get
from .command import helm
from .security import security_config


from .tool_registry import (
    KUBECTL_READ_ONLY_TOOLS,
    KUBECTL_ALL_TOOLS,
)

logger = logging.getLogger(__name__)


# Initialize FastMCP server
mcp = FastMCP("mcp-kubernetes")


def add_kubectl_tools():
    if security_config.readonly:
        # Register read-only functions
        for func in KUBECTL_READ_ONLY_TOOLS:
            logger.debug(f"Registering kubectl function: {func.__name__}")
            mcp.tool()(func)
    else:
        # Register all functions
        for func in KUBECTL_ALL_TOOLS:
            logger.debug(f"Registering kubectl function: {func.__name__}")
            mcp.tool()(func)


def server():
    """Run the MCP server."""
    parser = argparse.ArgumentParser(description="MCP Kubernetes Server")

    # command options
    parser.add_argument(
        "--disable-kubectl",
        action="store_true",
        help="Disable kubectl command execution",
    )
    parser.add_argument(
        "--disable-helm",
        action="store_true",
        help="Disable helm command execution",
    )

    # Transport options
    parser.add_argument(
        "--transport",
        type=str,
        choices=["stdio", "sse"],
        default="stdio",
        help="Transport mechanism to use (stdio or sse)",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=8000,
        help="Port to use for the server (only used with sse transport)",
    )

    # Security options
    parser.add_argument(
        "--readonly",
        action="store_true",
        default=False,
        help="Enable read-only mode (prevents write operations)",
    )
    parser.add_argument(
        "--allow-namespaces",
        type=str,
        default="",
        help="Comma-separated list of namespaces to allow (empty means all allowed)",
    )

    args = parser.parse_args()
    mcp.settings.port = args.port

    # Set security configuration
    security_config.readonly = args.readonly

    if args.allow_namespaces:
        security_config.allowed_namespaces = args.allow_namespaces

    # Setup Kubernetes client
    setup_client()

    # Setup tools
    if args.disable_kubectl:
        logger.debug("Registering kubectl API functions")
        mcp.tool()(apis)
        mcp.tool()(crds)
        mcp.tool()(get)
    else:
        logger.debug("Registering kubectl CLI functions")
        add_kubectl_tools()
    if not args.disable_helm:
        mcp.tool()(helm)

    # Run the server
    mcp.run(transport=args.transport)


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)
    server()
