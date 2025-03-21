package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/creack/pty"
	"golang.org/x/crypto/ssh/terminal"
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
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)

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

	// Send whatever we type into the PTY's stdin
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
	// Format the command description with bold and reset ANSI codes
	formattedCmd := fmt.Sprintf("%s%s%s", Bold, cmdDescription, Reset)

	// Print the prompt
	fmt.Printf("Are you sure you want to run %s? [y/N]: ", formattedCmd)

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	// Trim whitespace and convert to lowercase
	response = strings.TrimSpace(strings.ToLower(response))

	// Return true if the response is 'y' or 'yes'
	return response == "y" || response == "yes", nil
}
