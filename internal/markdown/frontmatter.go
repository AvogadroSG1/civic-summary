// Package markdown provides frontmatter parsing/injection, content sanitization,
// and cross-reference wikilink generation for Obsidian-compatible markdown.
package markdown

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontmatterDelimiter = "---"

// ParseFrontmatter splits a markdown document into YAML frontmatter and body content.
// Returns the parsed YAML as a map, the body content, and any error.
func ParseFrontmatter(content string) (map[string]interface{}, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, frontmatterDelimiter) {
		return nil, content, nil
	}

	// Find the closing delimiter.
	rest := content[len(frontmatterDelimiter):]
	closingIdx := strings.Index(rest[1:], "\n"+frontmatterDelimiter)
	if closingIdx == -1 {
		return nil, content, fmt.Errorf("unclosed frontmatter: no closing '---' found")
	}
	closingIdx++ // Adjust for the offset from rest[1:]

	yamlContent := rest[:closingIdx]
	body := strings.TrimSpace(rest[closingIdx+len("\n"+frontmatterDelimiter):])

	var fm map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, content, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	return fm, body, nil
}

// InjectFrontmatter creates a complete markdown document with YAML frontmatter.
func InjectFrontmatter(fm map[string]interface{}, body string) (string, error) {
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("marshaling frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(frontmatterDelimiter)
	sb.WriteString("\n")
	sb.Write(yamlBytes)
	sb.WriteString(frontmatterDelimiter)
	sb.WriteString("\n\n")
	sb.WriteString(body)

	return sb.String(), nil
}

// HasFrontmatter returns true if the content starts with a frontmatter delimiter.
func HasFrontmatter(content string) bool {
	return strings.HasPrefix(strings.TrimSpace(content), frontmatterDelimiter)
}

// RequiredFrontmatterKeys returns the keys that must be present in a valid summary.
var RequiredFrontmatterKeys = []string{"date", "author", "tags", "source", "meeting_date"}

// ValidateFrontmatter checks that all required keys are present.
func ValidateFrontmatter(fm map[string]interface{}) []string {
	var missing []string
	for _, key := range RequiredFrontmatterKeys {
		if _, ok := fm[key]; !ok {
			missing = append(missing, key)
		}
	}
	return missing
}
