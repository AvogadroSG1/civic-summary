package executor

import (
	"context"
	"fmt"
	"strings"
)

// ClaudeExecutor wraps Claude CLI operations for meeting analysis.
type ClaudeExecutor struct {
	commander Commander
	binary    string
}

// NewClaudeExecutor creates a new ClaudeExecutor.
func NewClaudeExecutor(commander Commander, binary string) *ClaudeExecutor {
	return &ClaudeExecutor{
		commander: commander,
		binary:    binary,
	}
}

// Analyze sends a prompt to Claude CLI and returns the response.
// Uses --print flag for non-interactive output.
func (c *ClaudeExecutor) Analyze(ctx context.Context, prompt string) (string, error) {
	result, err := c.commander.ExecuteWithStdin(ctx, prompt, c.binary, "--print")
	if err != nil {
		return "", fmt.Errorf("claude analysis: %w", err)
	}
	return strings.TrimSpace(result.Stdout), nil
}
