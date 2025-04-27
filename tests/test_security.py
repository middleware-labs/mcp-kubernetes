import unittest
from mcp_kubernetes.security import SecurityConfig


class TestSecurityConfig(unittest.TestCase):
    """Unit tests for SecurityConfig class."""

    def setUp(self):
        """Set up a fresh SecurityConfig instance for each test."""
        self.security_config = SecurityConfig()

    def test_readonly_default(self):
        """Test that readonly mode is False by default."""
        self.assertFalse(self.security_config.readonly)

    def test_readonly_setter(self):
        """Test setting readonly mode."""
        self.security_config.readonly = True
        self.assertTrue(self.security_config.readonly)

        self.security_config.readonly = False
        self.assertFalse(self.security_config.readonly)

    def test_allowed_namespaces_default(self):
        """Test that allowed_namespaces is empty by default."""
        self.assertEqual(self.security_config.allowed_namespaces, [])

    def test_allowed_namespaces_setter(self):
        """Test setting allowed_namespaces."""
        self.security_config.allowed_namespaces = "default,kube-system"
        self.assertEqual(
            self.security_config.allowed_namespaces, ["default", "kube-system"]
        )

        # Test with spaces
        self.security_config.allowed_namespaces = (
            "default, kube-system ,  app-namespace"
        )
        self.assertEqual(
            self.security_config.allowed_namespaces,
            ["default", "kube-system", "app-namespace"],
        )

        # Test with regex patterns - they should be separated into regex_patterns
        self.security_config.allowed_namespaces = "default,app-.*,test-\\d+"

        # Check that the regex patterns were correctly identified and compiled
        self.assertEqual(len(self.security_config.allowed_namespaces), 3)

    def test_is_namespace_allowed_no_restrictions(self):
        """Test namespace access with no restrictions."""
        # When no namespaces are explicitly allowed, all should be allowed
        self.assertTrue(self.security_config.is_namespace_allowed("default"))
        self.assertTrue(self.security_config.is_namespace_allowed("kube-system"))
        self.assertTrue(self.security_config.is_namespace_allowed("any-namespace"))

    def test_is_namespace_allowed_with_allowed_list(self):
        """Test namespace access with allowed list."""
        self.security_config.allowed_namespaces = "default,app-namespace"

        # Namespaces in the allowed list should be allowed
        self.assertTrue(self.security_config.is_namespace_allowed("default"))
        self.assertTrue(self.security_config.is_namespace_allowed("app-namespace"))

        # Namespaces not in the allowed list should be denied
        self.assertFalse(self.security_config.is_namespace_allowed("kube-system"))
        self.assertFalse(
            self.security_config.is_namespace_allowed("any-other-namespace")
        )

    def test_regex_namespace_patterns(self):
        """Test regex patterns for namespace matching."""
        # Set up with a mix of exact and regex patterns
        self.security_config.allowed_namespaces = "default,app-.*,test-\\d+,prod-[a-z]+"

        # Exact match should work
        self.assertTrue(self.security_config.is_namespace_allowed("default"))
        # Regex patterns should match appropriate namespaces
        self.assertTrue(self.security_config.is_namespace_allowed("app-frontend"))
        self.assertTrue(self.security_config.is_namespace_allowed("app-backend"))
        self.assertTrue(self.security_config.is_namespace_allowed("test-123"))
        self.assertTrue(self.security_config.is_namespace_allowed("prod-xyz"))

        # Non-matching namespaces should be denied
        self.assertFalse(self.security_config.is_namespace_allowed("app"))
        self.assertFalse(self.security_config.is_namespace_allowed("test"))
        self.assertFalse(self.security_config.is_namespace_allowed("prod-123"))
        self.assertFalse(self.security_config.is_namespace_allowed("staging"))

    def test_invalid_regex_patterns(self):
        """Test handling of invalid regex patterns."""
        # Set an invalid regex pattern
        self.security_config.allowed_namespaces = "default,[invalid"

        # The regex should be treated as a literal string
        self.assertTrue(self.security_config.is_namespace_allowed("default"))
        self.assertTrue(self.security_config.is_namespace_allowed("[invalid"))
        self.assertFalse(self.security_config.is_namespace_allowed("invalid"))


if __name__ == "__main__":
    unittest.main()
