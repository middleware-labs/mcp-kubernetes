import logging
import os
import shutil
import subprocess

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
    Check if kubectl is properly configured and can connect to the cluster.
    """
    try:
        # Run kubectl version with a short timeout to verify it's configured
        subprocess.run(
            ["kubectl", "version", "--request-timeout=1s"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        return True
    except subprocess.CalledProcessError:
        logger.error(
            "kubectl is not properly configured or cannot connect to the cluster."
        )
        return False
    except Exception as e:
        logger.error(f"Error validating kubectl configuration: {str(e)}")
        return False


def validate() -> bool:
    """
    Validate the configuration and environment,
    including required CLI tool availability.
    """

    return _validate_cli() and _validate_kubeconfig()
