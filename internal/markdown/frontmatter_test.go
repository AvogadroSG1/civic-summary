package markdown_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter_Valid(t *testing.T) {
	content := `---
date: 2025-02-05
author: Peter O'Connor
tags:
  - City-Council
  - Hagerstown
---

# Title

Body content here.`

	fm, body, err := markdown.ParseFrontmatter(content)
	require.NoError(t, err)

	// YAML unmarshals date-like strings as time.Time; check it exists and is non-nil.
	assert.NotNil(t, fm["date"])
	assert.Equal(t, "Peter O'Connor", fm["author"])
	assert.Contains(t, body, "# Title")
	assert.Contains(t, body, "Body content here.")
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	content := "# Just a title\n\nSome content."

	fm, body, err := markdown.ParseFrontmatter(content)
	require.NoError(t, err)
	assert.Nil(t, fm)
	assert.Contains(t, body, "# Just a title")
}

func TestParseFrontmatter_UnclosedDelimiter(t *testing.T) {
	content := "---\ndate: 2025\nauthor: Test\n"

	_, _, err := markdown.ParseFrontmatter(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unclosed frontmatter")
}

func TestInjectFrontmatter(t *testing.T) {
	fm := map[string]interface{}{
		"date":   "2025-02-05",
		"author": "Peter O'Connor",
	}
	body := "# Title\n\nContent."

	result, err := markdown.InjectFrontmatter(fm, body)
	require.NoError(t, err)
	assert.Contains(t, result, "---\n")
	assert.Contains(t, result, "date: \"2025-02-05\"")
	assert.Contains(t, result, "# Title")
}

func TestHasFrontmatter(t *testing.T) {
	assert.True(t, markdown.HasFrontmatter("---\ndate: 2025\n---\n"))
	assert.True(t, markdown.HasFrontmatter("  ---\ndate: 2025\n---\n"))
	assert.False(t, markdown.HasFrontmatter("# No frontmatter"))
	assert.False(t, markdown.HasFrontmatter(""))
}

func TestValidateFrontmatter_AllPresent(t *testing.T) {
	fm := map[string]interface{}{
		"date":         "2025-02-05",
		"author":       "Peter O'Connor",
		"tags":         []string{"tag1"},
		"source":       "https://youtube.com/watch?v=abc",
		"meeting_date": "2025-02-04",
	}
	missing := markdown.ValidateFrontmatter(fm)
	assert.Empty(t, missing)
}

func TestValidateFrontmatter_MissingKeys(t *testing.T) {
	fm := map[string]interface{}{
		"date": "2025-02-05",
	}
	missing := markdown.ValidateFrontmatter(fm)
	assert.Len(t, missing, 4) // author, tags, source, meeting_date
	assert.Contains(t, missing, "author")
	assert.Contains(t, missing, "tags")
	assert.Contains(t, missing, "source")
	assert.Contains(t, missing, "meeting_date")
}
