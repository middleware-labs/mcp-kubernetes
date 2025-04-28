import unittest
from unittest import mock

from mcp_kubernetes.tool_registry import (
    KUBECTL_READONLY,
    KUBECTL_RW,
    KUBECTL_ADMIN,
    kubectl_func_register,
    get_kubectl_functions_by_label,
    get_all_kubectl_registered_functions,
    KUBECTL_FUNCTION_REGISTRY,
)


class TestToolRegistry(unittest.TestCase):
    """Unit tests for the tool registry module."""

    def setUp(self):
        """Set up test environment."""
        # Clear the registry before each test
        KUBECTL_FUNCTION_REGISTRY.clear()

    def test_kubectl_func_register_decorator(self):
        """Test that kubectl_func_register decorator correctly registers functions."""

        # Define a test function and decorate it
        @kubectl_func_register(KUBECTL_READONLY)
        def test_func_readonly(command: str) -> str:
            return f"Read: {command}"

        @kubectl_func_register(KUBECTL_RW)
        def test_func_rw(command: str) -> str:
            return f"Write: {command}"

        @kubectl_func_register(KUBECTL_ADMIN)
        def test_func_admin(command: str) -> str:
            return f"Admin: {command}"

        # Check that functions are correctly registered
        self.assertIn("test_func_readonly", KUBECTL_FUNCTION_REGISTRY)
        self.assertIn("test_func_rw", KUBECTL_FUNCTION_REGISTRY)
        self.assertIn("test_func_admin", KUBECTL_FUNCTION_REGISTRY)

        # Check that labels are correctly assigned
        self.assertEqual(
            KUBECTL_FUNCTION_REGISTRY["test_func_readonly"]["label"], KUBECTL_READONLY
        )
        self.assertEqual(KUBECTL_FUNCTION_REGISTRY["test_func_rw"]["label"], KUBECTL_RW)
        self.assertEqual(
            KUBECTL_FUNCTION_REGISTRY["test_func_admin"]["label"], KUBECTL_ADMIN
        )

        # Check that function objects are correctly stored
        self.assertEqual(
            KUBECTL_FUNCTION_REGISTRY["test_func_readonly"]["function"]("test"),
            "Read: test",
        )
        self.assertEqual(
            KUBECTL_FUNCTION_REGISTRY["test_func_rw"]["function"]("test"), "Write: test"
        )
        self.assertEqual(
            KUBECTL_FUNCTION_REGISTRY["test_func_admin"]["function"]("test"),
            "Admin: test",
        )

    def test_get_kubectl_functions_by_label(self):
        """Test get_kubectl_functions_by_label returns correct functions."""

        # Define test functions
        @kubectl_func_register(KUBECTL_READONLY)
        def test_func_readonly1(command: str) -> str:
            return f"Read1: {command}"

        @kubectl_func_register(KUBECTL_READONLY)
        def test_func_readonly2(command: str) -> str:
            return f"Read2: {command}"

        @kubectl_func_register(KUBECTL_RW)
        def test_func_rw(command: str) -> str:
            return f"Write: {command}"

        # Get readonly functions
        readonly_functions = get_kubectl_functions_by_label(KUBECTL_READONLY)
        rw_functions = get_kubectl_functions_by_label(KUBECTL_RW)
        admin_functions = get_kubectl_functions_by_label(KUBECTL_ADMIN)

        # Check correct number of functions returned
        self.assertEqual(len(readonly_functions), 2)
        self.assertEqual(len(rw_functions), 1)
        self.assertEqual(len(admin_functions), 0)

        # Check that function objects are correct
        self.assertIn(test_func_readonly1, readonly_functions)
        self.assertIn(test_func_readonly2, readonly_functions)
        self.assertIn(test_func_rw, rw_functions)

    def test_get_all_kubectl_registered_functions(self):
        """Test get_all_kubectl_registered_functions returns all functions."""

        # Define test functions
        @kubectl_func_register(KUBECTL_READONLY)
        def test_func_readonly(command: str) -> str:
            return f"Read: {command}"

        @kubectl_func_register(KUBECTL_RW)
        def test_func_rw(command: str) -> str:
            return f"Write: {command}"

        @kubectl_func_register(KUBECTL_ADMIN)
        def test_func_admin(command: str) -> str:
            return f"Admin: {command}"

        # Get all functions
        all_functions = get_all_kubectl_registered_functions()

        # Check correct number of functions returned
        self.assertEqual(len(all_functions), 3)

        # Check that function objects are correct
        function_names = [func.__name__ for func in all_functions]
        self.assertIn("test_func_readonly", function_names)
        self.assertIn("test_func_rw", function_names)
        self.assertIn("test_func_admin", function_names)


if __name__ == "__main__":
    unittest.main()
