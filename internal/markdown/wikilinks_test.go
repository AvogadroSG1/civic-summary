package markdown_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddWikilinks_NoMatchingFiles(t *testing.T) {
	content := "The meeting on February 04, 2025 discussed several topics."
	cfg := markdown.CrossReferenceConfig{
		BaseDir:         "/nonexistent",
		FilenamePattern: "Hagerstown-City-Council-{{.MeetingDate}}-Citizen-Summary",
		SelfDate:        time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	result := markdown.AddWikilinks(content, cfg)
	// No files exist, so no wikilinks should be added.
	assert.Equal(t, content, result)
}

func TestAddWikilinks_SkipsSelfDate(t *testing.T) {
	selfDate := time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC)
	content := "The meeting on February 04, 2025 was held."
	cfg := markdown.CrossReferenceConfig{
		BaseDir:         "/nonexistent",
		FilenamePattern: "Test-{{.MeetingDate}}-Summary",
		SelfDate:        selfDate,
	}

	result := markdown.AddWikilinks(content, cfg)
	// Self-date should not be linked.
	assert.NotContains(t, result, "[[")
}

func TestAddWikilinks_WithExistingFile(t *testing.T) {
	// Create a temp directory with a fake summary.
	tmpDir := t.TempDir()
	dateDir := filepath.Join(tmpDir, "20250204")
	require.NoError(t, os.MkdirAll(dateDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dateDir, "Test-2025-02-04-Summary.md"),
		[]byte("# Test"), 0o644,
	))

	content := "Reference to February 04, 2025 meeting."
	cfg := markdown.CrossReferenceConfig{
		BaseDir:         tmpDir,
		FilenamePattern: "Test-{{.MeetingDate}}-Summary",
		SelfDate:        time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	result := markdown.AddWikilinks(content, cfg)
	assert.Contains(t, result, "[[Test-2025-02-04-Summary|February 04, 2025]]")
}

func TestAddWikilinks_SequencedFile(t *testing.T) {
	// A single sequenced file (sequence 1) should resolve via glob fallback.
	tmpDir := t.TempDir()
	dateDir := filepath.Join(tmpDir, "20250204")
	require.NoError(t, os.MkdirAll(dateDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dateDir, "Test-2025-02-04-Summary-1.md"),
		[]byte("# Test"), 0o644,
	))

	content := "Reference to February 04, 2025 meeting."
	cfg := markdown.CrossReferenceConfig{
		BaseDir:         tmpDir,
		FilenamePattern: "Test-{{.MeetingDate}}-Summary",
		SelfDate:        time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	result := markdown.AddWikilinks(content, cfg)
	assert.Contains(t, result, "[[Test-2025-02-04-Summary-1|February 04, 2025]]")
}

func TestAddWikilinks_MultipleSequencedFiles(t *testing.T) {
	// When multiple sequenced files exist, link to the first (sequence 1).
	tmpDir := t.TempDir()
	dateDir := filepath.Join(tmpDir, "20250204")
	require.NoError(t, os.MkdirAll(dateDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dateDir, "Test-2025-02-04-Summary-1.md"),
		[]byte("# Test 1"), 0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dateDir, "Test-2025-02-04-Summary-2.md"),
		[]byte("# Test 2"), 0o644,
	))

	content := "Reference to February 04, 2025 meeting."
	cfg := markdown.CrossReferenceConfig{
		BaseDir:         tmpDir,
		FilenamePattern: "Test-{{.MeetingDate}}-Summary",
		SelfDate:        time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	result := markdown.AddWikilinks(content, cfg)
	// Should link to sequence 1 (alphabetically first).
	assert.Contains(t, result, "[[Test-2025-02-04-Summary-1|February 04, 2025]]")
	assert.NotContains(t, result, "Summary-2")
}

func TestAddWikilinks_MultipleReferences(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two target summaries.
	for _, date := range []string{"20250204", "20250121"} {
		dateDir := filepath.Join(tmpDir, date)
		require.NoError(t, os.MkdirAll(dateDir, 0o755))
		isoDate := date[:4] + "-" + date[4:6] + "-" + date[6:8]
		require.NoError(t, os.WriteFile(
			filepath.Join(dateDir, "Test-"+isoDate+"-Summary.md"),
			[]byte("# Test"), 0o644,
		))
	}

	content := "Discussed at February 04, 2025 and January 21, 2025."
	cfg := markdown.CrossReferenceConfig{
		BaseDir:         tmpDir,
		FilenamePattern: "Test-{{.MeetingDate}}-Summary",
		SelfDate:        time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
	}

	result := markdown.AddWikilinks(content, cfg)
	assert.Contains(t, result, "[[Test-2025-02-04-Summary|February 04, 2025]]")
	assert.Contains(t, result, "[[Test-2025-01-21-Summary|January 21, 2025]]")
}
