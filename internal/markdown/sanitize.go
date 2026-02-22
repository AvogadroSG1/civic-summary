package markdown

import (
	"regexp"
	"strings"
)

// claudeMetaPatterns are regex patterns that detect Claude's
// meta-commentary that should not appear in the final summary.
var claudeMetaPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^Based on the`),
	regexp.MustCompile(`(?i)^I'll `),
	regexp.MustCompile(`(?i)^I will `),
	regexp.MustCompile(`(?i)^Let me `),
	regexp.MustCompile(`(?i)^Here's `),
	regexp.MustCompile(`(?i)^Here is `),
	regexp.MustCompile(`(?i)^Below is `),
	regexp.MustCompile(`(?i)^The following `),
	regexp.MustCompile(`(?i)^This document `),
	regexp.MustCompile(`(?i)^I've analyzed `),
	regexp.MustCompile(`(?i)^I have analyzed `),
	regexp.MustCompile(`(?i)^After reviewing `),
}

// HasClaudeMetaCommentary checks if the content contains any Claude meta-commentary
// patterns that indicate the model is "talking about" the document rather than
// producing the document itself.
func HasClaudeMetaCommentary(content string) bool {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		for _, pattern := range claudeMetaPatterns {
			if pattern.MatchString(trimmed) {
				return true
			}
		}
	}
	return false
}

// Sanitize removes Claude meta-commentary lines from the beginning of content.
// It strips any lines before the first frontmatter delimiter that match
// meta-commentary patterns.
func Sanitize(content string) string {
	content = strings.TrimSpace(content)

	// If content already starts with frontmatter, nothing to strip.
	if strings.HasPrefix(content, "---") {
		return content
	}

	// Find where the frontmatter starts and discard everything before it.
	idx := strings.Index(content, "\n---")
	if idx != -1 {
		return strings.TrimSpace(content[idx+1:])
	}

	return content
}
