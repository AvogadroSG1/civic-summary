package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/config"
	"github.com/AvogadroSG1/civic-summary/internal/domain"
)

// QuarantineService manages failed meetings for later retry.
type QuarantineService struct {
	cfg *config.Config
}

// NewQuarantineService creates a new QuarantineService.
func NewQuarantineService(cfg *config.Config) *QuarantineService {
	return &QuarantineService{cfg: cfg}
}

// Quarantine adds a failed meeting to the quarantine directory.
func (s *QuarantineService) Quarantine(body domain.Body, meeting domain.Meeting, errMsg string, transcriptPath string, partialOutput string) error {
	qDir := filepath.Join(s.cfg.QuarantineDir(body), meeting.VideoID)
	if err := os.MkdirAll(qDir, 0o755); err != nil {
		return fmt.Errorf("creating quarantine dir: %w", err)
	}

	// Write metadata.
	entry := domain.QuarantineEntry{
		VideoID:       meeting.VideoID,
		MeetingDate:   meeting.ISODate(),
		BodySlug:      body.Slug,
		Error:         errMsg,
		QuarantinedAt: time.Now(),
		RetryCount:    0,
	}

	metadataPath := filepath.Join(qDir, "metadata.json")
	if err := writeJSON(metadataPath, entry); err != nil {
		return fmt.Errorf("writing quarantine metadata: %w", err)
	}

	// Preserve transcript if available.
	if transcriptPath != "" {
		src, err := os.ReadFile(transcriptPath)
		if err == nil {
			_ = os.WriteFile(filepath.Join(qDir, "transcript.srt"), src, 0o644)
		}
	}

	// Preserve partial output if available.
	if partialOutput != "" {
		_ = os.WriteFile(filepath.Join(qDir, "output.md"), []byte(partialOutput), 0o644)
	}

	// Update manifest.
	if err := s.addToManifest(body, meeting.VideoID, meeting.ISODate()); err != nil {
		slog.Warn("failed to update quarantine manifest", "error", err)
	}

	slog.Info("meeting quarantined",
		"video_id", meeting.VideoID,
		"body", body.Slug,
		"error", errMsg,
	)

	return nil
}

// ListQuarantined returns all quarantined entries for a body.
func (s *QuarantineService) ListQuarantined(body domain.Body) ([]domain.QuarantineEntry, error) {
	qDir := s.cfg.QuarantineDir(body)
	entries, err := os.ReadDir(qDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading quarantine dir: %w", err)
	}

	var result []domain.QuarantineEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metadataPath := filepath.Join(qDir, entry.Name(), "metadata.json")
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			continue
		}
		var qe domain.QuarantineEntry
		if err := json.Unmarshal(data, &qe); err != nil {
			continue
		}
		result = append(result, qe)
	}

	return result, nil
}

// IncrementRetry increases the retry count for a quarantined entry.
func (s *QuarantineService) IncrementRetry(body domain.Body, videoID string) error {
	metadataPath := filepath.Join(s.cfg.QuarantineDir(body), videoID, "metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("reading metadata: %w", err)
	}

	var entry domain.QuarantineEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("parsing metadata: %w", err)
	}

	entry.RetryCount++
	return writeJSON(metadataPath, entry)
}

// Remove removes a meeting from quarantine (typically after successful retry).
func (s *QuarantineService) Remove(body domain.Body, videoID string) error {
	qDir := filepath.Join(s.cfg.QuarantineDir(body), videoID)
	if err := os.RemoveAll(qDir); err != nil {
		return fmt.Errorf("removing quarantine entry: %w", err)
	}

	// Update manifest.
	return s.removeFromManifest(body, videoID)
}

// addToManifest adds an entry to the quarantine manifest.
func (s *QuarantineService) addToManifest(body domain.Body, videoID, meetingDate string) error {
	manifest, err := s.loadManifest(body)
	if err != nil {
		manifest = domain.NewQuarantineManifest()
	}

	manifest.Items[videoID] = domain.QuarantineManifestEntry{
		MeetingDate: meetingDate,
		BodySlug:    body.Slug,
		Added:       time.Now(),
	}

	return s.saveManifest(body, manifest)
}

// removeFromManifest removes an entry from the quarantine manifest.
func (s *QuarantineService) removeFromManifest(body domain.Body, videoID string) error {
	manifest, err := s.loadManifest(body)
	if err != nil {
		return nil // No manifest, nothing to remove.
	}
	delete(manifest.Items, videoID)
	return s.saveManifest(body, manifest)
}

// loadManifest reads the quarantine manifest from disk.
func (s *QuarantineService) loadManifest(body domain.Body) (*domain.QuarantineManifest, error) {
	path := filepath.Join(s.cfg.QuarantineDir(body), "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest domain.QuarantineManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// saveManifest writes the quarantine manifest to disk.
func (s *QuarantineService) saveManifest(body domain.Body, manifest *domain.QuarantineManifest) error {
	if err := os.MkdirAll(s.cfg.QuarantineDir(body), 0o755); err != nil {
		return err
	}
	path := filepath.Join(s.cfg.QuarantineDir(body), "manifest.json")
	return writeJSON(path, manifest)
}

// writeJSON marshals and writes JSON to a file.
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
