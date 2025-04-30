import logging
import os
import shutil

from .config import config

logger = logging.getLogger(__name__)


def _is_cli_installed(cli_name: str) -> bool:
    """
    Check if a CLI tool is installed and available in the system PATH.
    """
    return shutil.which(cli_name) is not None


def _validate_cli() -> bool:
    """
    Check if the required CLI tools are installed.
    """
    required_tools = ["kubectl"]
    if not config.disable_helm:
        required_tools.append("helm")

    for tool in required_tools:
        if not _is_cli_installed(tool):
            logger.error(f"{tool} is not installed or not found in PATH.")
            return False

    # TODO: Should we check the versions of the required CLIs?

    return True


def _validate_kubeconfig() -> bool:
    """
    Check if the kubeconfig file is present and valid.
    """

    kubeconfig_path = os.getenv("KUBECONFIG")
    if not kubeconfig_path:
        logger.error("KUBECONFIG environment variable is not set.")
        return False
    if not os.path.exists(kubeconfig_path):
        logger.error(f"Kubeconfig file not found at {kubeconfig_path}.")
        return False
    if not os.access(kubeconfig_path, os.R_OK):
        logger.error(f"Kubeconfig file at {kubeconfig_path} is not readable.")
        return False

    return True


def validate() -> bool:
    """
    Validate the configuration and environment,
    including required CLI tool availability.
    """

    return _validate_cli() and _validate_kubeconfig()
