package domain

import "strings"

// Summary is a value object representing the final markdown document
// produced by the analysis pipeline.
type Summary struct {
	Content     string
	Path        string
	Frontmatter map[string]interface{}
}

// WordCount returns the number of words in the summary content.
func (s Summary) WordCount() int {
	if s.Content == "" {
		return 0
	}
	return len(strings.Fields(s.Content))
}
