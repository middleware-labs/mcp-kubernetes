package security

import (
	"strings"
	"testing"
)

func TestValidatorReadOnly(t *testing.T) {
	secConfig := NewSecurityConfig()
	secConfig.ReadOnly = true
	validator := NewValidator(secConfig)

	// Test read operation
	err := validator.ValidateCommand("kubectl get pods", CommandTypeKubectl)
	if err != nil {
		t.Errorf("Read operation should be allowed in read-only mode: %v", err)
	}

	// Test write operation
	err = validator.ValidateCommand("kubectl delete pods mypod", CommandTypeKubectl)
	if err == nil {
		t.Error("Write operation should not be allowed in read-only mode")
	}
}

func TestValidatorNamespaceRestriction(t *testing.T) {
	secConfig := NewSecurityConfig()
	secConfig.SetAllowedNamespaces("allowed-ns,another-ns")
	validator := NewValidator(secConfig)

	// Test allowed namespace
	err := validator.ValidateCommand("kubectl get pods -n allowed-ns", CommandTypeKubectl)
	if err != nil {
		t.Errorf("Allowed namespace should be accessible: %v", err)
	}

	// Test disallowed namespace
	err = validator.ValidateCommand("kubectl get pods -n disallowed-ns", CommandTypeKubectl)
	if err == nil {
		t.Error("Disallowed namespace should not be accessible")
	}

	// Test all namespaces restriction
	err = validator.ValidateCommand("kubectl get pods --all-namespaces", CommandTypeKubectl)
	if err == nil {
		t.Error("All namespaces should not be accessible when restrictions are in place")
	}
}

func TestNamespaceHandling(t *testing.T) {
	// Test namespace handling via public ValidateCommand method

	// Setup validator with namespace restrictions
	secConfig := NewSecurityConfig()
	secConfig.SetAllowedNamespaces("test-ns,another-ns,default")
	validator := NewValidator(secConfig)

	// Test cases for namespace handling
	tests := []struct {
		command   string
		shouldErr bool
		errMsg    string
	}{
		{"kubectl get pods -n test-ns", false, ""},
		{"kubectl get pods --namespace=another-ns", false, ""},
		{"kubectl get pod/mypod", false, ""},
		{"kubectl get pods -n disallowed-ns", true, "denied by security configuration"},
		{"kubectl get pods --all-namespaces", true, "restricted by security configuration"},
		{"kubectl get pods -A", true, "restricted by security configuration"},
	}

	for _, tc := range tests {
		err := validator.ValidateCommand(tc.command, CommandTypeKubectl)

		if tc.shouldErr && err == nil {
			t.Errorf("ValidateCommand(%q) should have failed", tc.command)
		} else if !tc.shouldErr && err != nil {
			t.Errorf("ValidateCommand(%q) should have succeeded, got: %v", tc.command, err)
		} else if err != nil && tc.shouldErr && !strings.Contains(err.Error(), tc.errMsg) {
			t.Errorf("ValidateCommand(%q) error message mismatch, got: %v, want: %v", tc.command, err, tc.errMsg)
		}
	}
}

func TestReadOperationsValidation(t *testing.T) {
	// Test read operations validation through public API
	secConfig := NewSecurityConfig()
	secConfig.ReadOnly = true
	validator := NewValidator(secConfig)

	// Test cases for read operations
	tests := []struct {
		command     string
		commandType string
		shouldErr   bool
	}{
		{"kubectl get pods", CommandTypeKubectl, false},
		{"kubectl describe pod mypod", CommandTypeKubectl, false},
		{"kubectl delete pod mypod", CommandTypeKubectl, true},
		{"kubectl create namespace test", CommandTypeKubectl, true},
		{"helm list", CommandTypeHelm, false},
		{"helm status release", CommandTypeHelm, false},
		{"helm install chart", CommandTypeHelm, true},
		{"helm uninstall release", CommandTypeHelm, true},
		{"cilium status", CommandTypeCilium, false},
		{"cilium endpoint list", CommandTypeCilium, false},
		{"cilium install", CommandTypeCilium, true},
	}

	for _, tc := range tests {
		err := validator.ValidateCommand(tc.command, tc.commandType)

		if tc.shouldErr && err == nil {
			t.Errorf("ValidateCommand(%q, %q) should have failed", tc.command, tc.commandType)
		} else if !tc.shouldErr && err != nil {
			t.Errorf("ValidateCommand(%q, %q) should have succeeded, got: %v", tc.command, tc.commandType, err)
		}
	}
}

func TestValidateCommand(t *testing.T) {
	// Comprehensive test with multiple security configurations
	testCases := []struct {
		name        string
		readonly    bool
		namespaces  string
		command     string
		commandType string
		shouldErr   bool
	}{
		{"Read operation in readonly mode", true, "", "kubectl get pods", CommandTypeKubectl, false},
		{"Write operation in readonly mode", true, "", "kubectl delete pods", CommandTypeKubectl, true},

		{"Command in allowed namespace", false, "ns1,ns2", "kubectl get pods -n ns1", CommandTypeKubectl, false},
		{"Command in disallowed namespace", false, "ns1,ns2", "kubectl get pods -n ns3", CommandTypeKubectl, true},

		{"All namespaces restricted", false, "ns1,ns2", "kubectl get pods --all-namespaces", CommandTypeKubectl, true},

		// Combined restrictions
		{"Read op in allowed ns with readonly", true, "ns1", "kubectl get pods -n ns1", CommandTypeKubectl, false},
		{"Read op in disallowed ns with readonly", true, "ns1", "kubectl get pods -n ns2", CommandTypeKubectl, true},
		{"Write op in allowed ns with readonly", true, "ns1", "kubectl delete pods -n ns1", CommandTypeKubectl, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secConfig := NewSecurityConfig()
			secConfig.ReadOnly = tc.readonly
			if tc.namespaces != "" {
				secConfig.SetAllowedNamespaces(tc.namespaces)
			}

			validator := NewValidator(secConfig)
			err := validator.ValidateCommand(tc.command, tc.commandType)

			if tc.shouldErr && err == nil {
				t.Errorf("ValidateCommand should have failed")
			} else if !tc.shouldErr && err != nil {
				t.Errorf("ValidateCommand should have succeeded, got: %v", err)
			}
		})
	}
}
