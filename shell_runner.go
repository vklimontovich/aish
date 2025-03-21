package main

import (
	"bytes"
	"fmt"
	"github.com/creack/pty"
	"io"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

// ANSI Colors
const (
	Red    = "\033[91m"
	Green  = "\033[92m"
	Yellow = "\033[93m"
	Cyan   = "\033[96m"
	Bold   = "\033[1m\033[4m"
	Reset  = "\033[0m"
)

// func main() {
//     cmd := `echo "What is your name? "; read name && echo "You answered: $name"`
//     stdout, stderr, exitCode := runShellCommand(cmd)
//     fmt.Printf("\n---\nExit code: %d\nstderr:\n%v\nstdout:\n%v\n", exitCode, stderr, stdout)
// }

func runShellCommand(cmd string) (string, string, int) {
	fmt.Printf("ℹ️ Running command: %s\n", cmd)

	execCmd := exec.Command("sh", "-c", cmd)

	// Start the command with a pty so that read behaves interactively
	ptmx, err := pty.Start(execCmd)
	if err != nil {
		return "", fmt.Sprintf("Failed to start command with PTY: %v", err), 1
	}
	defer ptmx.Close()

	// We won't set our own terminal to raw mode. That way, hitting Enter
	// sends a proper newline (in cooked mode) that read can detect.

	// Buffers to capture the final output
	var stdoutBuf, stderrBuf bytes.Buffer

	// We 'tee' everything from the PTY into our stdout buffer
	reader := io.TeeReader(ptmx, &stdoutBuf)

	// Send PTY output to our console, so we see the prompt
	go func() {
		_, _ = io.Copy(os.Stdout, reader)
	}()

	// Send whatever we type into the PTY’s stdin
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// Wait for the command to exit
	err = execCmd.Wait()
	if err != nil {
		return stdoutBuf.String(), fmt.Sprintf("Command failed: %v", err), 1
	}
	return stdoutBuf.String(), stderrBuf.String(), 0
}

func format(pattern string, data interface{}) (string, error) {
	tmpl, err := template.New("myTemplate").Delims("{", "}").Parse(pattern)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func promptConfirm(cmdDescription string) (bool, error) {
	shellSnippet, err := format(`
      read -p "Are you sure you want to run {.bold}{.cmd}{.reset}? [y/N] " answer
      # Echo the user input so Go can parse it
      echo "USER_ANSWER=$answer"
    `, map[string]interface{}{"cmd": cmdDescription, "bold": Bold, "reset": Reset})
	if err != nil {
		exitWithError("Failed to format shell snippet: %v", err)
	}
	cmd := exec.Command("sh", "-c", shellSnippet)

	// Create PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return false, fmt.Errorf("failed to start PTY: %w", err)
	}
	defer ptmx.Close()

	// We capture all PTY output into stdoutBuf
	var stdoutBuf bytes.Buffer
	// Use TeeReader so that everything from PTY is:
	//   1) Copied into stdoutBuf
	//   2) Also printed to user’s screen
	reader := io.TeeReader(ptmx, &stdoutBuf)

	// Display anything the shell writes (prompt, echo, etc.) in real-time
	go func() {
		_, _ = io.Copy(os.Stdout, reader)
	}()

	// Forward anything user types from our real stdin to the PTY
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// Wait for shell to finish
	err = cmd.Wait()
	if err != nil {
		return false, fmt.Errorf("shell command failed: %w", err)
	}

	// Now parse stdoutBuf for "USER_ANSWER=..."
	lines := strings.Split(stdoutBuf.String(), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "USER_ANSWER=") {
			answer := strings.TrimPrefix(line, "USER_ANSWER=")
			answer = strings.ToLower(strings.TrimSpace(answer))
			return (answer == "y"), nil
		}
	}

	// If we never saw a "USER_ANSWER=", treat as not confirmed
	return false, nil
}
