import unittest
import os
import shutil
from unittest import mock

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

    @mock.patch("os.path.exists")
    @mock.patch("os.access")
    @mock.patch.dict(os.environ, {}, clear=True)
    def test_validate_kubeconfig_no_env_var(self, mock_access, mock_exists):
        """Test _validate_kubeconfig when KUBECONFIG environment variable is not set."""
        self.assertFalse(_validate_kubeconfig())
        # These shouldn't be called if env var is not set
        mock_exists.assert_not_called()
        mock_access.assert_not_called()

    @mock.patch("os.path.exists")
    @mock.patch("os.access")
    @mock.patch.dict(os.environ, {"KUBECONFIG": "/path/to/kubeconfig"})
    def test_validate_kubeconfig_file_not_found(self, mock_access, mock_exists):
        """Test _validate_kubeconfig when kubeconfig file doesn't exist."""
        mock_exists.return_value = False
        self.assertFalse(_validate_kubeconfig())
        mock_exists.assert_called_once_with("/path/to/kubeconfig")
        mock_access.assert_not_called()

    @mock.patch("os.path.exists")
    @mock.patch("os.access")
    @mock.patch.dict(os.environ, {"KUBECONFIG": "/path/to/kubeconfig"})
    def test_validate_kubeconfig_file_not_readable(self, mock_access, mock_exists):
        """Test _validate_kubeconfig when kubeconfig file is not readable."""
        mock_exists.return_value = True
        mock_access.return_value = False
        self.assertFalse(_validate_kubeconfig())
        mock_exists.assert_called_once_with("/path/to/kubeconfig")
        mock_access.assert_called_once_with("/path/to/kubeconfig", os.R_OK)

    @mock.patch("os.path.exists")
    @mock.patch("os.access")
    @mock.patch.dict(os.environ, {"KUBECONFIG": "/path/to/kubeconfig"})
    def test_validate_kubeconfig_valid(self, mock_access, mock_exists):
        """Test _validate_kubeconfig with valid kubeconfig."""
        mock_exists.return_value = True
        mock_access.return_value = True
        self.assertTrue(_validate_kubeconfig())
        mock_exists.assert_called_once_with("/path/to/kubeconfig")
        mock_access.assert_called_once_with("/path/to/kubeconfig", os.R_OK)

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
