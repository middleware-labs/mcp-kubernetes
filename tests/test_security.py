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

    def test_denied_namespaces_default(self):
        """Test that denied_namespaces is empty by default."""
        self.assertEqual(self.security_config.denied_namespaces, [])

    def test_denied_namespaces_setter(self):
        """Test setting denied_namespaces."""
        self.security_config.denied_namespaces = "kube-system,kube-public"
        self.assertEqual(
            self.security_config.denied_namespaces, ["kube-system", "kube-public"]
        )

        # Test with spaces
        self.security_config.denied_namespaces = (
            "kube-system, kube-public ,  secret-namespace"
        )
        self.assertEqual(
            self.security_config.denied_namespaces,
            ["kube-system", "kube-public", "secret-namespace"],
        )

    def test_is_namespace_allowed_no_restrictions(self):
        """Test namespace access with no restrictions."""
        # When no namespaces are explicitly allowed or denied, all should be allowed
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

    def test_is_namespace_allowed_with_denied_list(self):
        """Test namespace access with denied list."""
        self.security_config.denied_namespaces = "kube-system,secret-namespace"

        # Namespaces in the denied list should be denied
        self.assertFalse(self.security_config.is_namespace_allowed("kube-system"))
        self.assertFalse(self.security_config.is_namespace_allowed("secret-namespace"))

        # Namespaces not in the denied list should be allowed
        self.assertTrue(self.security_config.is_namespace_allowed("default"))
        self.assertTrue(self.security_config.is_namespace_allowed("app-namespace"))

    def test_denied_list_takes_precedence(self):
        """Test that denied_namespaces takes precedence over allowed_namespaces."""
        self.security_config.allowed_namespaces = "default,kube-system,app-namespace"
        self.security_config.denied_namespaces = "kube-system,secret-namespace"

        # Namespaces in both allowed and denied lists should be denied
        self.assertFalse(self.security_config.is_namespace_allowed("kube-system"))

        # Namespaces only in allowed list should be allowed
        self.assertTrue(self.security_config.is_namespace_allowed("default"))
        self.assertTrue(self.security_config.is_namespace_allowed("app-namespace"))

        # Namespaces only in denied list should be denied
        self.assertFalse(self.security_config.is_namespace_allowed("secret-namespace"))

        # Namespaces in neither list should be denied (since allowed list is not empty)
        self.assertFalse(
            self.security_config.is_namespace_allowed("any-other-namespace")
        )


if __name__ == "__main__":
    unittest.main()
