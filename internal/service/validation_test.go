package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(wd, "..", "..", "testdata", "fixtures", name))
	require.NoError(t, err)
	return string(content)
}

func testBody() domain.Body {
	return domain.Body{
		Slug:       "hagerstown",
		Name:       "Hagerstown City Council",
		Tags:       []string{"City-Council"},
		FooterText: "This citizen summary was created from the official meeting video and transcript.",
	}
}

func TestValidationService_ValidSummary(t *testing.T) {
	content := loadFixture(t, "valid-summary.md")
	svc := service.NewValidationService()

	result := svc.Validate(content, testBody())

	assert.True(t, result.IsValid(), "Expected valid but got errors: %v", result.Errors())
	// May have warnings but no errors.
	for _, issue := range result.Issues {
		assert.Equal(t, domain.SeverityWarning, issue.Severity, "unexpected error: %s", issue.Message)
	}
}

func TestValidationService_MissingFrontmatter(t *testing.T) {
	content := "# Title\n\nNo frontmatter here."
	svc := service.NewValidationService()

	result := svc.Validate(content, testBody())

	assert.False(t, result.IsValid())
	assert.True(t, result.HasErrors())
}

func TestValidationService_MissingSections(t *testing.T) {
	content := `---
date: 2025-02-05
author: Peter O'Connor
tags:
  - City-Council
source: https://youtube.com/watch?v=abc
meeting_date: 2025-02-04
---

# Title

## 1. Updates

Some updates.

## 3. Actions Taken

Some actions.

## Conclusion

Done.`

	// Pad with enough words to pass word count.
	content += "\n\n" + strings.Repeat("word ", 500)

	svc := service.NewValidationService()
	result := svc.Validate(content, testBody())

	// Should have errors for missing sections 2, 4, 5.
	errors := result.Errors()
	assert.True(t, len(errors) >= 3, "Expected at least 3 errors for missing sections, got %d", len(errors))
}

func TestValidationService_ShortSummary(t *testing.T) {
	content := `---
date: 2025-02-05
author: Peter O'Connor
tags:
  - City-Council
source: https://youtube.com/watch?v=abc
meeting_date: 2025-02-04
---

# Title

## 1. Updates
Short.
## 2. Citizen Comments
None.
## 3. Actions Taken
None.
## 4. Input Requested from Council
None.
## 5. Critical Discussions
None.
## Conclusion
Done.`

	svc := service.NewValidationService()
	result := svc.Validate(content, testBody())

	assert.False(t, result.IsValid())
	hasWordCountError := false
	for _, e := range result.Errors() {
		if strings.Contains(e.Message, "too short") {
			hasWordCountError = true
		}
	}
	assert.True(t, hasWordCountError, "Expected word count error")
}

func TestValidationService_ClaudeMetaCommentary(t *testing.T) {
	content := `Based on the transcript, here is the summary.

# Title

## 1. Updates
` + strings.Repeat("word ", 500)

	svc := service.NewValidationService()
	result := svc.Validate(content, testBody())

	assert.False(t, result.IsValid())
}

func TestValidationService_TagsWithSpaces(t *testing.T) {
	content := `---
date: 2025-02-05
author: Peter O'Connor
tags:
  - City Council
  - Hagerstown
source: https://youtube.com/watch?v=abc
meeting_date: 2025-02-04
---

# Title
## 1. Updates
## 2. Citizen Comments
## 3. Actions Taken
## 4. Input Requested from Council
## 5. Critical Discussions

` + strings.Repeat("word ", 500) + `

## Conclusion

*This citizen summary was created from the official meeting video and transcript.*`

	svc := service.NewValidationService()
	result := svc.Validate(content, testBody())

	hasTagError := false
	for _, e := range result.Errors() {
		if strings.Contains(e.Message, "spaces") {
			hasTagError = true
		}
	}
	assert.True(t, hasTagError, "Expected tag spacing error")
}
