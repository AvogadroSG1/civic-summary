package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/markdown"
)

const (
	minSummaryWords         = 500
	warningSummaryWordCount = 1000
)

// Required section headings (## 1. through ## 5.)
var requiredSections = []string{
	"## 1.",
	"## 2.",
	"## 3.",
	"## 4.",
	"## 5.",
}

var timestampPattern = regexp.MustCompile(`\[\d{1,2}:\d{2}:\d{2}`)

// ValidationService validates that summaries meet quality requirements.
type ValidationService struct{}

// NewValidationService creates a new ValidationService.
func NewValidationService() *ValidationService {
	return &ValidationService{}
}

// Validate performs comprehensive validation on a summary.
func (s *ValidationService) Validate(content string, body domain.Body) *domain.ValidationResult {
	result := &domain.ValidationResult{}

	s.validateFrontmatter(content, result)
	s.validateStructure(content, body, result)
	s.validateContent(content, result)
	s.validateMetaCommentary(content, result)

	return result
}

// validateFrontmatter checks YAML frontmatter presence and required fields.
func (s *ValidationService) validateFrontmatter(content string, result *domain.ValidationResult) {
	if !markdown.HasFrontmatter(content) {
		result.AddError("missing frontmatter: document must start with '---'")
		return
	}

	fm, _, err := markdown.ParseFrontmatter(content)
	if err != nil {
		result.AddError("invalid frontmatter: %s", err)
		return
	}

	missing := markdown.ValidateFrontmatter(fm)
	for _, key := range missing {
		result.AddError("missing required frontmatter key: %s", key)
	}

	// Validate tags format (hyphens, not spaces).
	if tags, ok := fm["tags"]; ok {
		if tagList, ok := tags.([]interface{}); ok {
			for _, tag := range tagList {
				if tagStr, ok := tag.(string); ok {
					if strings.Contains(tagStr, " ") {
						result.AddError("tag contains spaces (use hyphens): %q", tagStr)
					}
				}
			}
		}
	}
}

// validateStructure checks for required sections and headings.
func (s *ValidationService) validateStructure(content string, body domain.Body, result *domain.ValidationResult) {
	for _, section := range requiredSections {
		if !strings.Contains(content, section) {
			result.AddError("missing required section: %s", section)
		}
	}

	// Check for main title heading.
	lines := strings.Split(content, "\n")
	hasTitle := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			hasTitle = true
			break
		}
	}
	if !hasTitle {
		result.AddError("missing main title (# heading)")
	}

	// Check for conclusion section.
	if !strings.Contains(content, "## Conclusion") {
		result.AddWarning("missing Conclusion section")
	}

	// Check for attribution footer.
	footerText := "citizen summary was created"
	if body.FooterText != "" {
		footerText = body.FooterText
	}
	if !strings.Contains(strings.ToLower(content), strings.ToLower(footerText)) {
		result.AddWarning("missing attribution footer text")
	}
}

// validateContent checks word count and timestamp presence.
func (s *ValidationService) validateContent(content string, result *domain.ValidationResult) {
	wordCount := len(strings.Fields(content))

	if wordCount < minSummaryWords {
		result.AddError("summary too short (%d words, minimum %d)", wordCount, minSummaryWords)
	} else if wordCount < warningSummaryWordCount {
		result.AddWarning("summary is short (%d words, recommend %d+)", wordCount, warningSummaryWordCount)
	}

	if !timestampPattern.MatchString(content) {
		result.AddWarning("no timestamps found (expected [HH:MM:SS] format)")
	}
}

// validateMetaCommentary checks for Claude meta-commentary in the output.
func (s *ValidationService) validateMetaCommentary(content string, result *domain.ValidationResult) {
	// Only check the first few lines (before frontmatter should be clean).
	if !markdown.HasFrontmatter(content) {
		// If no frontmatter, the whole start is suspect.
		if markdown.HasClaudeMetaCommentary(content) {
			result.AddError("contains Claude meta-commentary (output must start with frontmatter)")
		}
	}

	// Also check after frontmatter for embedded commentary.
	_, body, err := markdown.ParseFrontmatter(content)
	if err == nil && body != "" {
		firstLine := strings.TrimSpace(strings.Split(body, "\n")[0])
		for _, pattern := range []string{"Based on", "I'll ", "I will ", "Let me ", "Here's ", "Here is "} {
			if strings.HasPrefix(firstLine, pattern) {
				result.AddError("body starts with Claude meta-commentary: %q", fmt.Sprintf("%.50s...", firstLine))
				break
			}
		}
	}
}
