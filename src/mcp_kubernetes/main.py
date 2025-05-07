# -*- coding: utf-8 -*-
import argparse
import sys
from fastmcp import FastMCP
import logging

from .args_validator import validate
from .command import helm, cilium
from .config import config
from .kubeclient import setup_client
from .tool_registry import (
    KUBECTL_READ_ONLY_TOOLS,
    KUBECTL_RW_TOOLS,
    KUBECTL_ADMIN_TOOLS,
)

logger = logging.getLogger(__name__)


# Initialize FastMCP server
mcp = FastMCP("mcp-kubernetes")


def add_kubectl_tools():
    # Register read-only functions
    for func in KUBECTL_READ_ONLY_TOOLS:
        logger.debug(f"Registering kubectl function: {func.__name__}")
        mcp.tool()(func)

    # Register rw and admin functions
    if not config.security_config.readonly:
        for func in KUBECTL_RW_TOOLS + KUBECTL_ADMIN_TOOLS:
            logger.debug(f"Registering kubectl function: {func.__name__}")
            mcp.tool()(func)


def server():
    """Run the MCP server."""
    parser = argparse.ArgumentParser(description="MCP Kubernetes Server")

    # command options
    parser.add_argument(
        "--additional-tools",
        type=str,
        default="",
        help="Comma-separated list of additional tools to support (kubectl is always enabled). Available: helm,cilium",
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

    # timeout configuration
    parser.add_argument(
        "--timeout",
        type=int,
        default=60,
        help="Timeout for command execution in seconds, default is 60s",
    )

    args = parser.parse_args()
    mcp.settings.port = args.port

    # Set security configuration
    config.security_config.readonly = args.readonly

    if args.allow_namespaces:
        config.security_config.allowed_namespaces = args.allow_namespaces

    # Set timeout configuration
    if args.timeout:
        config.timeout = args.timeout

    # Parse and register supported tools
    if args.additional_tools:
        # Split the comma-separated list and strip whitespace
        requested_tools = [
            tool.strip().lower()
            for tool in args.additional_tools.split(",")
            if tool.strip()
        ]

        # Validate that all requested tools are supported
        unsupported_tools = [
            tool for tool in requested_tools if tool not in config.available_tools
        ]
        if unsupported_tools:
            logger.error(
                f"Error: Unsupported tool(s) specified: {', '.join(unsupported_tools)}"
            )
            logger.error(
                f"Supported tools are: {', '.join(sorted(config.available_tools))}"
            )
            sys.exit(1)

        # Add valid tools to the additional_tools set
        for tool in requested_tools:
            config.additional_tools.add(tool)
            logger.info(f"Enabling support for tool: {tool}")

    # Note: needs to be done after all the configurations are set
    if not validate():
        logger.error("Validation failed. Exiting.")
        sys.exit(1)

    # Setup Kubernetes client
    setup_client()

    # Setup kubectl tools
    # TODO: need a further discussion on using k8s sdk or kubectl, comment out these codes as they are duplicated with kubectl
    add_kubectl_tools()

    # Setup additional tools based on configuration
    if "helm" in config.additional_tools:
        logger.debug("Registering helm function")
        mcp.tool(
            "Run-helm-command",
            "Run helm command and get result, The command should start with helm",
        )(helm)

    if "cilium" in config.additional_tools:
        logger.debug("Registering cilium function")
        mcp.tool(
            "Run-cilium-command",
            "Run cilium command and get result, The command should start with cilium",
        )(cilium)

    # Run the server
    mcp.run(transport=args.transport)


if __name__ == "__main__":
    logging.basicConfig(level=logging.DEBUG)
    server()
