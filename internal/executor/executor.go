// Package executor provides a mockable interface for shell-out commands.
// All interactions with external tools (yt-dlp, whisper, claude) go through
// this interface, enabling unit tests without real binaries.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandResult holds the output of an executed command.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Commander is the interface for executing external commands.
// Implementations include the real OsCommander and test mocks.
type Commander interface {
	// Execute runs a command with the given arguments and returns the result.
	Execute(ctx context.Context, name string, args ...string) (*CommandResult, error)

	// ExecuteWithStdin runs a command with stdin input.
	ExecuteWithStdin(ctx context.Context, stdin string, name string, args ...string) (*CommandResult, error)
}

// OsCommander executes real OS commands via os/exec.
type OsCommander struct{}

// NewOsCommander creates a new OsCommander.
func NewOsCommander() *OsCommander {
	return &OsCommander{}
}

// Execute runs a command and captures stdout/stderr.
func (o *OsCommander) Execute(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	return o.ExecuteWithStdin(ctx, "", name, args...)
}

// ExecuteWithStdin runs a command with stdin input.
func (o *OsCommander) ExecuteWithStdin(ctx context.Context, stdin string, name string, args ...string) (*CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	err := cmd.Run()
	result := &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, fmt.Errorf("command %q exited with code %d: %s", name, result.ExitCode, stderr.String())
		}
		return result, fmt.Errorf("executing %q: %w", name, err)
	}

	return result, nil
}
