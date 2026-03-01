package service

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
)

// IndexService generates and updates the master meeting index file.
type IndexService struct {
	cfg *config.Config
}

// NewIndexService creates a new IndexService.
func NewIndexService(cfg *config.Config) *IndexService {
	return &IndexService{cfg: cfg}
}

// UpdateIndex regenerates the index.md file listing all finalized meetings for a body.
func (s *IndexService) UpdateIndex(body domain.Body) error {
	finalizedDir := s.cfg.FinalizedDir(body)
	indexPath := filepath.Join(finalizedDir, "index.md")

	entries, err := os.ReadDir(finalizedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading finalized dir: %w", err)
	}

	// Collect meeting folders (YYYYMMDD format).
	var folders []string
	for _, entry := range entries {
		if entry.IsDir() && len(entry.Name()) == 8 {
			folders = append(folders, entry.Name())
		}
	}

	// Sort descending (most recent first).
	sort.Sort(sort.Reverse(sort.StringSlice(folders)))

	// Collect all summary entries across all folders.
	type indexEntry struct {
		summaryName string
		dateStr     string
	}
	var allEntries []indexEntry

	for _, folder := range folders {
		// Find all summary .md files in this folder.
		mdFiles, err := filepath.Glob(filepath.Join(finalizedDir, folder, "*.md"))
		if err != nil || len(mdFiles) == 0 {
			continue
		}

		// Format the date from folder name.
		dateStr := fmt.Sprintf("%s-%s-%s", folder[:4], folder[4:6], folder[6:8])

		// List every .md file in the folder (handles same-date disambiguation).
		sort.Strings(mdFiles)
		for _, mdFile := range mdFiles {
			summaryName := strings.TrimSuffix(filepath.Base(mdFile), ".md")
			allEntries = append(allEntries, indexEntry{summaryName, dateStr})
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s - Meeting Index\n\n", body.Name)
	fmt.Fprintf(&sb, "*%d meetings processed*\n\n", len(allEntries))

	for _, entry := range allEntries {
		fmt.Fprintf(&sb, "- [[%s|%s]]\n", entry.summaryName, entry.dateStr)
	}

	if err := os.WriteFile(indexPath, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing index: %w", err)
	}

	slog.Info("index updated",
		"body", body.Slug,
		"meetings", len(folders),
		"path", indexPath,
	)

	return nil
}
