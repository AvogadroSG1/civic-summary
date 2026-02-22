package executor_test

import (
	"context"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOsCommander_Execute_Success(t *testing.T) {
	cmd := executor.NewOsCommander()
	result, err := cmd.Execute(context.Background(), "echo", "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", result.Stdout)
	assert.Equal(t, 0, result.ExitCode)
}

func TestOsCommander_Execute_Failure(t *testing.T) {
	cmd := executor.NewOsCommander()
	_, err := cmd.Execute(context.Background(), "false")
	assert.Error(t, err)
}

func TestOsCommander_Execute_NotFound(t *testing.T) {
	cmd := executor.NewOsCommander()
	_, err := cmd.Execute(context.Background(), "nonexistent-command-xyz")
	assert.Error(t, err)
}

func TestMockCommander_RecordsCalls(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.OnCommand("test-cmd", &executor.CommandResult{Stdout: "ok"}, nil)

	result, err := mock.Execute(context.Background(), "test-cmd")
	require.NoError(t, err)
	assert.Equal(t, "ok", result.Stdout)
	assert.Equal(t, []string{"test-cmd"}, mock.Calls)
}

func TestMockCommander_WithArgs(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.OnCommand("git status", &executor.CommandResult{Stdout: "clean"}, nil)

	result, err := mock.Execute(context.Background(), "git", "status")
	require.NoError(t, err)
	assert.Equal(t, "clean", result.Stdout)
}

func TestMockCommander_DefaultResult(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{Stdout: "default"}

	result, err := mock.Execute(context.Background(), "anything")
	require.NoError(t, err)
	assert.Equal(t, "default", result.Stdout)
}
