package markdown_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/markdown"
	"github.com/stretchr/testify/assert"
)

func TestHasClaudeMetaCommentary(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"clean frontmatter", "---\ndate: 2025\n---\n# Title", false},
		{"based on", "Based on the transcript, here is the summary.\n---\n", true},
		{"I'll create", "I'll create a comprehensive summary.\n---\n", true},
		{"Let me", "Let me analyze this meeting.\n---\n", true},
		{"Here's the", "Here's the citizen summary.\n---\n", true},
		{"case insensitive", "based on the transcript", true},
		{"empty", "", false},
		{"normal content", "The council met on Tuesday.", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, markdown.HasClaudeMetaCommentary(tt.content))
		})
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"already clean",
			"---\ndate: 2025\n---\n# Title",
			"---\ndate: 2025\n---\n# Title",
		},
		{
			"with preamble",
			"Here's the summary:\n---\ndate: 2025\n---\n# Title",
			"---\ndate: 2025\n---\n# Title",
		},
		{
			"multi-line preamble",
			"I'll create this.\nBased on the transcript.\n---\ndate: 2025\n---\n# Title",
			"---\ndate: 2025\n---\n# Title",
		},
		{
			"no frontmatter at all",
			"Just some text without frontmatter",
			"Just some text without frontmatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, markdown.Sanitize(tt.input))
		})
	}
}
