package executor

import (
	"context"
	"fmt"
)

// MockCommander is a test double for Commander that returns pre-configured responses.
type MockCommander struct {
	// Responses maps "command arg1 arg2..." to the result to return.
	Responses map[string]*CommandResult
	// Errors maps command keys to errors.
	Errors map[string]error
	// Calls records all commands that were executed, in order.
	Calls []string
	// DefaultResult is returned when no specific response is configured.
	DefaultResult *CommandResult
}

// NewMockCommander creates a new MockCommander.
func NewMockCommander() *MockCommander {
	return &MockCommander{
		Responses: make(map[string]*CommandResult),
		Errors:    make(map[string]error),
	}
}

// OnCommand sets up a response for a specific command invocation.
func (m *MockCommander) OnCommand(key string, result *CommandResult, err error) {
	m.Responses[key] = result
	if err != nil {
		m.Errors[key] = err
	}
}

// Execute records the call and returns the pre-configured response.
func (m *MockCommander) Execute(_ context.Context, name string, args ...string) (*CommandResult, error) {
	return m.ExecuteWithStdin(context.Background(), "", name, args...)
}

// ExecuteWithStdin records the call and returns the pre-configured response.
func (m *MockCommander) ExecuteWithStdin(_ context.Context, _ string, name string, args ...string) (*CommandResult, error) {
	key := name
	if len(args) > 0 {
		key = fmt.Sprintf("%s %s", name, joinArgs(args))
	}
	m.Calls = append(m.Calls, key)

	if result, ok := m.Responses[key]; ok {
		return result, m.Errors[key]
	}

	// Try matching just the command name.
	if result, ok := m.Responses[name]; ok {
		return result, m.Errors[name]
	}

	if m.DefaultResult != nil {
		return m.DefaultResult, nil
	}

	return &CommandResult{}, nil
}

func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}
