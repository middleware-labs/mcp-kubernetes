package command

import (
	"strings"
	"testing"
)

func TestExecBasicCommand(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	output, err := sp.Exec("echo hello")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("Expected 'hello' in output, got: %q", output)
	}
}

func TestRunWithArgs(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	output, err := sp.Run("hello world")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected 'hello world' in output, got: %q", output)
	}
}

func TestRunWithEmptyArgs(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	output, err := sp.Run("")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if output == "" {
		t.Errorf("Expected non-empty output, got empty string")
	}
}

func TestExecWithQuotedArgs(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	output, err := sp.Exec("echo \"hello world\"")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected 'hello world' in output, got: %q", output)
	}
}

func TestExecWithComplexQuotes(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	output, err := sp.Exec("echo \"quoted argument with 'nested quotes'\"")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(output, "quoted argument with 'nested quotes'") {
		t.Errorf("Expected quoted content in output, got: %q", output)
	}
}

func TestExecWithInvalidCommand(t *testing.T) {
	sp := NewShellProcess("nonexistentcommand", 5)
	_, err := sp.Exec("nonexistentcommand")

	if err == nil {
		t.Errorf("Expected error for nonexistent command, got none")
	}
}

func TestRunWithQuotedArgs(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	output, err := sp.Run("\"hello world\"")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(output, "hello world") {
		t.Errorf("Expected 'hello world' in output, got: %q", output)
	}
}

func TestRunWithMixedQuotes(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	// Test with a mix of single and double quotes
	output, err := sp.Run("\"hello world\" 'another quoted part'")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !strings.Contains(output, "hello world") || !strings.Contains(output, "another quoted part") {
		t.Errorf("Expected both quoted parts in output, got: %q", output)
	}
}

func TestParsingComplexCommand(t *testing.T) {
	sp := NewShellProcess("echo", 5)
	// This simulates a more complex kubectl command with quoted arguments
	output, err := sp.Exec("echo -n ns \"with spaces\" --flag=\"quoted value\"")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that quotes are properly handled
	if !strings.Contains(output, "with spaces") || !strings.Contains(output, "quoted value") {
		t.Errorf("Expected properly parsed quoted arguments, got: %q", output)
	}
}

func TestExecWithEmptyCommand(t *testing.T) {
	sp := NewShellProcess("", 5)
	output, err := sp.Exec("")

	if err != nil {
		t.Fatalf("Expected no error for empty command, got: %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output for empty command, got: %q", output)
	}
}

func TestStripNewlines(t *testing.T) {
	// Using echo with the -e flag to interpret escape sequences
	sp := NewShellProcess("echo", 5)
	sp.StripNewlines = true
	output, err := sp.Run("-n hello")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check that the output is "hello" without newlines
	expected := "hello"
	if output != expected {
		t.Errorf("Expected %q after stripping newlines, got: %q", expected, output)
	}
}

func TestTimeout(t *testing.T) {
	// This test uses 'sleep' which should be available on all Unix/Linux systems
	sp := NewShellProcess("sleep", 1) // 1 second timeout
	_, err := sp.Exec("sleep 2")      // Sleep for 2 seconds

	if err == nil {
		t.Errorf("Expected timeout error, got none")
	}
}

func TestReturnErrOutput(t *testing.T) {
	sp := NewShellProcess("cat", 5)
	sp.ReturnErrOutput = true
	output, err := sp.Exec("cat nonexistentfile")

	if err != nil {
		t.Errorf("Expected no error when ReturnErrOutput=true, got: %v", err)
	}
	if output == "" {
		t.Errorf("Expected error output, got empty string")
	}

	sp.ReturnErrOutput = false
	_, err = sp.Exec("cat nonexistentfile")

	if err == nil {
		t.Errorf("Expected error when ReturnErrOutput=false, got none")
	}
}
