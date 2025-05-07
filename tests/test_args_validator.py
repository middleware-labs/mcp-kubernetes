import unittest
import os
import shutil
from unittest import mock
import subprocess

from mcp_kubernetes.args_validator import (
    _is_cli_installed,
    _validate_cli,
    _validate_kubeconfig,
    validate,
)
from mcp_kubernetes.config import config


class TestArgsValidator(unittest.TestCase):
    """Unit tests for args_validator module."""

    def setUp(self):
        """Set up test environment."""
        # Save original config state
        self.original_disable_helm = config.disable_helm

    def tearDown(self):
        """Restore original config state after tests."""
        config.disable_helm = self.original_disable_helm

    @mock.patch("shutil.which")
    def test_is_cli_installed(self, mock_which):
        """Test _is_cli_installed function."""
        # Test when CLI is installed
        mock_which.return_value = "/usr/bin/kubectl"
        self.assertTrue(_is_cli_installed("kubectl"))

        # Test when CLI is not installed
        mock_which.return_value = None
        self.assertFalse(_is_cli_installed("kubectl"))

        # Verify correct CLI name was passed to shutil.which
        mock_which.assert_called_with("kubectl")

    @mock.patch("mcp_kubernetes.args_validator._is_cli_installed")
    def test_validate_cli(self, mock_is_cli_installed):
        """Test _validate_cli function."""
        # Test when all tools are installed
        mock_is_cli_installed.return_value = True
        self.assertTrue(_validate_cli())

        # Test when kubectl is not installed
        mock_is_cli_installed.side_effect = lambda tool: tool != "kubectl"
        self.assertFalse(_validate_cli())

        # Test when helm is not installed but required
        mock_is_cli_installed.side_effect = lambda tool: tool != "helm"
        config.disable_helm = False
        self.assertFalse(_validate_cli())

        # Test when helm is not installed but disabled
        config.disable_helm = True
        self.assertTrue(_validate_cli())

    @mock.patch("subprocess.run")
    def test_validate_kubeconfig_successful(self, mock_run):
        """Test _validate_kubeconfig when kubectl version runs successfully."""
        # Setup mock to return a successful CompletedProcess
        mock_run.return_value = subprocess.CompletedProcess(
            args=["kubectl", "version", "--request-timeout=1s"],
            returncode=0,
            stdout=b"Client Version: v1.25.0\nServer Version: v1.26.0\n",
            stderr=b"",
        )

        self.assertTrue(_validate_kubeconfig())
        mock_run.assert_called_once_with(
            ["kubectl", "version", "--request-timeout=1s"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    @mock.patch("subprocess.run")
    def test_validate_kubeconfig_command_failed(self, mock_run):
        """Test _validate_kubeconfig when kubectl version fails."""
        # Setup mock to raise CalledProcessError
        mock_run.side_effect = subprocess.CalledProcessError(
            returncode=1,
            cmd=["kubectl", "version", "--request-timeout=1s"],
            output=b"",
            stderr=b"Error from server: connection refused",
        )

        self.assertFalse(_validate_kubeconfig())
        mock_run.assert_called_once()

    @mock.patch("subprocess.run")
    def test_validate_kubeconfig_exception(self, mock_run):
        """Test _validate_kubeconfig when an unexpected exception occurs."""
        # Setup mock to raise a generic exception
        mock_run.side_effect = Exception("Some unexpected error")

        self.assertFalse(_validate_kubeconfig())
        mock_run.assert_called_once()

    @mock.patch("mcp_kubernetes.args_validator._validate_cli")
    @mock.patch("mcp_kubernetes.args_validator._validate_kubeconfig")
    def test_validate(self, mock_validate_kubeconfig, mock_validate_cli):
        """Test validate function."""
        # Test when both validations pass
        mock_validate_cli.return_value = True
        mock_validate_kubeconfig.return_value = True
        self.assertTrue(validate())

        # Test when CLI validation fails
        mock_validate_cli.return_value = False
        mock_validate_kubeconfig.return_value = True
        self.assertFalse(validate())

        # Test when kubeconfig validation fails
        mock_validate_cli.return_value = True
        mock_validate_kubeconfig.return_value = False
        self.assertFalse(validate())

        # Test when both validations fail
        mock_validate_cli.return_value = False
        mock_validate_kubeconfig.return_value = False
        self.assertFalse(validate())


if __name__ == "__main__":
    unittest.main()
