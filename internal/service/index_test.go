package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexService_UpdateIndex(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		OutputDir: tmpDir,
		Bodies: map[string]domain.Body{
			"test": {
				Slug:         "test",
				Name:         "Test Body",
				OutputSubdir: "Test",
			},
		},
	}

	body, _ := cfg.GetBody("test")
	finalizedDir := cfg.FinalizedDir(body)

	// Create some meeting folders with summary files.
	for _, date := range []string{"20250204", "20250121", "20250107"} {
		dir := filepath.Join(finalizedDir, date)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		iso := date[:4] + "-" + date[4:6] + "-" + date[6:]
		require.NoError(t, os.WriteFile(
			filepath.Join(dir, "Summary-"+iso+".md"),
			[]byte("# Summary"), 0o644,
		))
	}

	svc := service.NewIndexService(cfg)
	err := svc.UpdateIndex(body)
	require.NoError(t, err)

	// Read the generated index.
	indexPath := filepath.Join(finalizedDir, "index.md")
	content, err := os.ReadFile(indexPath)
	require.NoError(t, err)

	indexStr := string(content)
	assert.Contains(t, indexStr, "Test Body")
	assert.Contains(t, indexStr, "3 meetings")
	assert.Contains(t, indexStr, "2025-02-04")
	assert.Contains(t, indexStr, "2025-01-21")
	assert.Contains(t, indexStr, "2025-01-07")
}

func TestIndexService_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		OutputDir: tmpDir,
		Bodies: map[string]domain.Body{
			"test": {
				Slug:         "test",
				OutputSubdir: "Test",
			},
		},
	}

	body, _ := cfg.GetBody("test")
	svc := service.NewIndexService(cfg)
	err := svc.UpdateIndex(body)
	// Should not error even if dir doesn't exist.
	assert.NoError(t, err)
}
