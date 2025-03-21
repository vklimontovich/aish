package main

import (
	"github.com/creack/pty"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestRunShellCommand(t *testing.T) {
	// Test case 1: Simple echo command
	t.Run("EchoCommand", func(t *testing.T) {
		stdout, stderr, exitCode := runShellCommand(`echo "XYZ"`)
		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
		}
		if stderr != "" {
			t.Errorf("Expected no stderr, got %s", stderr)
		}
		expectedOutput := "XYZ"
		if strings.TrimSpace(stdout) != expectedOutput {
			t.Errorf("Expected stdout %q, got %q", expectedOutput, strings.TrimSpace(stdout))
		}
	})

	// Test case 2: Conditional test for interactive command
	if os.Getenv("TEST_TTY") != "" {
		t.Run("InteractiveCommand", func(t *testing.T) {
			// Simulate user input by echoing the input to the command
			cmd := exec.Command("sh", "-c", `echo "What is your name? "; read name && echo "You answered: $name"`)
			stdout, stderr, exitCode := runShellCommand(cmd)
			if exitCode != 0 {
				t.Errorf("Expected exit code 0, got %d", exitCode)
			}
			if stderr != "" {
				t.Errorf("Expected no stderr, got %s", stderr)
			}
			expectedOutput := "You answered: John Doe\n"
			if strings.TrimSpace(stdout) != expectedOutput {
				t.Errorf("Expected stdout %q, got %q", expectedOutput, strings.TrimSpace(stdout))
			}
		})
	}
}
