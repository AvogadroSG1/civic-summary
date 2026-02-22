package domain_test

import (
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestValidationResult_NoIssues(t *testing.T) {
	r := &domain.ValidationResult{}
	assert.True(t, r.IsValid())
	assert.False(t, r.HasErrors())
	assert.Empty(t, r.Errors())
	assert.Empty(t, r.Warnings())
}

func TestValidationResult_WithErrors(t *testing.T) {
	r := &domain.ValidationResult{}
	r.AddError("missing field: %s", "date")
	r.AddError("invalid format")

	assert.False(t, r.IsValid())
	assert.True(t, r.HasErrors())
	assert.Len(t, r.Errors(), 2)
	assert.Empty(t, r.Warnings())
}

func TestValidationResult_WithWarnings(t *testing.T) {
	r := &domain.ValidationResult{}
	r.AddWarning("short content")

	assert.True(t, r.IsValid())
	assert.False(t, r.HasErrors())
	assert.Empty(t, r.Errors())
	assert.Len(t, r.Warnings(), 1)
}

func TestValidationResult_Mixed(t *testing.T) {
	r := &domain.ValidationResult{}
	r.AddError("critical issue")
	r.AddWarning("minor issue")

	assert.False(t, r.IsValid())
	assert.True(t, r.HasErrors())
	assert.Len(t, r.Errors(), 1)
	assert.Len(t, r.Warnings(), 1)
	assert.Len(t, r.Issues, 2)
}

func TestValidationIssue_String(t *testing.T) {
	errIssue := domain.ValidationIssue{
		Severity: domain.SeverityError,
		Message:  "test error",
	}
	assert.Equal(t, "[ERROR] test error", errIssue.String())

	warnIssue := domain.ValidationIssue{
		Severity: domain.SeverityWarning,
		Message:  "test warning",
	}
	assert.Equal(t, "[WARNING] test warning", warnIssue.String())
}
