package domain

import "fmt"

// ValidationSeverity indicates whether a validation issue is a hard failure or advisory.
type ValidationSeverity int

const (
	SeverityError   ValidationSeverity = iota // Hard fail — summary is rejected
	SeverityWarning                           // Advisory — summary passes with notes
)

// ValidationIssue represents a single validation problem.
type ValidationIssue struct {
	Severity ValidationSeverity
	Message  string
}

func (v ValidationIssue) String() string {
	prefix := "ERROR"
	if v.Severity == SeverityWarning {
		prefix = "WARNING"
	}
	return fmt.Sprintf("[%s] %s", prefix, v.Message)
}

// ValidationResult aggregates all issues found during validation.
type ValidationResult struct {
	Issues []ValidationIssue
}

// AddError appends a hard-fail error.
func (r *ValidationResult) AddError(msg string, args ...interface{}) {
	r.Issues = append(r.Issues, ValidationIssue{
		Severity: SeverityError,
		Message:  fmt.Sprintf(msg, args...),
	})
}

// AddWarning appends an advisory warning.
func (r *ValidationResult) AddWarning(msg string, args ...interface{}) {
	r.Issues = append(r.Issues, ValidationIssue{
		Severity: SeverityWarning,
		Message:  fmt.Sprintf(msg, args...),
	})
}

// HasErrors returns true if any hard-fail errors exist.
func (r *ValidationResult) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only the error-severity issues.
func (r *ValidationResult) Errors() []ValidationIssue {
	var errors []ValidationIssue
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			errors = append(errors, issue)
		}
	}
	return errors
}

// Warnings returns only the warning-severity issues.
func (r *ValidationResult) Warnings() []ValidationIssue {
	var warnings []ValidationIssue
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			warnings = append(warnings, issue)
		}
	}
	return warnings
}

// IsValid returns true if the result has no errors (warnings are acceptable).
func (r *ValidationResult) IsValid() bool {
	return !r.HasErrors()
}
