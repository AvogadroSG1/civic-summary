package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
)

const minTranscriptWords = 500

// TranscriptionService obtains meeting transcripts via captions or Whisper fallback.
type TranscriptionService struct {
	ytdlp   *executor.YtDlpExecutor
	whisper *executor.WhisperExecutor
}

// NewTranscriptionService creates a new TranscriptionService.
func NewTranscriptionService(ytdlp *executor.YtDlpExecutor, whisper *executor.WhisperExecutor) *TranscriptionService {
	return &TranscriptionService{ytdlp: ytdlp, whisper: whisper}
}

// Transcribe obtains a transcript for a meeting, trying captions first
// then falling back to Whisper audio transcription.
func (s *TranscriptionService) Transcribe(ctx context.Context, meeting domain.Meeting, outputDir string) (domain.Transcript, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return domain.Transcript{}, fmt.Errorf("creating output dir: %w", err)
	}

	// Try captions first (fast path).
	transcript, err := s.tryCaptions(ctx, meeting, outputDir)
	if err == nil && !transcript.IsEmpty() {
		slog.Info("obtained transcript via captions",
			"video_id", meeting.VideoID,
			"words", transcript.WordCount(),
		)
		return transcript, nil
	}

	slog.Info("captions unavailable, falling back to whisper",
		"video_id", meeting.VideoID,
	)

	// Whisper fallback.
	transcript, err = s.tryWhisper(ctx, meeting, outputDir)
	if err != nil {
		return domain.Transcript{}, fmt.Errorf("whisper fallback failed: %w", err)
	}

	slog.Info("obtained transcript via whisper",
		"video_id", meeting.VideoID,
		"words", transcript.WordCount(),
	)

	return transcript, nil
}

// ValidateTranscript checks that a transcript meets minimum quality requirements.
func (s *TranscriptionService) ValidateTranscript(transcript domain.Transcript) error {
	if transcript.IsEmpty() {
		return fmt.Errorf("transcript is empty")
	}
	wc := transcript.WordCount()
	if wc < minTranscriptWords {
		return fmt.Errorf("transcript too short (%d words, need %d)", wc, minTranscriptWords)
	}
	return nil
}

// tryCaptions attempts to download auto-generated captions.
func (s *TranscriptionService) tryCaptions(ctx context.Context, meeting domain.Meeting, outputDir string) (domain.Transcript, error) {
	srtPath, err := s.ytdlp.DownloadCaptions(ctx, meeting.VideoID, outputDir)
	if err != nil {
		return domain.Transcript{}, err
	}
	if srtPath == "" {
		return domain.Transcript{}, fmt.Errorf("no captions available")
	}

	content, err := os.ReadFile(srtPath)
	if err != nil {
		return domain.Transcript{}, fmt.Errorf("reading captions: %w", err)
	}

	// Rename from .en.srt to .srt for consistency.
	finalPath := filepath.Join(outputDir, meeting.VideoID+".srt")
	if srtPath != finalPath {
		if err := os.Rename(srtPath, finalPath); err != nil {
			// Non-fatal: keep original path.
			finalPath = srtPath
		}
	}

	return domain.Transcript{
		Content: string(content),
		Path:    finalPath,
		Source:  domain.TranscriptSourceCaptions,
	}, nil
}

// tryWhisper downloads audio and transcribes with whisper.cpp.
func (s *TranscriptionService) tryWhisper(ctx context.Context, meeting domain.Meeting, outputDir string) (domain.Transcript, error) {
	if s.whisper == nil {
		return domain.Transcript{}, fmt.Errorf("whisper not configured")
	}

	// Download audio.
	audioPath := filepath.Join(outputDir, meeting.VideoID+".mp3")
	if err := s.ytdlp.DownloadAudio(ctx, meeting.VideoID, audioPath); err != nil {
		return domain.Transcript{}, fmt.Errorf("downloading audio: %w", err)
	}

	// Transcribe.
	outputBase := filepath.Join(outputDir, meeting.VideoID)
	srtPath, err := s.whisper.Transcribe(ctx, audioPath, outputBase)
	if err != nil {
		return domain.Transcript{}, fmt.Errorf("transcribing audio: %w", err)
	}

	content, err := os.ReadFile(srtPath)
	if err != nil {
		return domain.Transcript{}, fmt.Errorf("reading transcript: %w", err)
	}

	return domain.Transcript{
		Content: strings.TrimSpace(string(content)),
		Path:    srtPath,
		Source:  domain.TranscriptSourceWhisper,
	}, nil
}
