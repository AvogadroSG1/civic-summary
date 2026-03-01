package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// WhisperExecutor wraps OpenAI whisper CLI operations for audio transcription.
type WhisperExecutor struct {
	commander Commander
	binary    string
	model     string
}

// NewWhisperExecutor creates a new WhisperExecutor.
func NewWhisperExecutor(commander Commander, binary string, model string) *WhisperExecutor {
	return &WhisperExecutor{
		commander: commander,
		binary:    binary,
		model:     model,
	}
}

// Transcribe runs OpenAI whisper on an audio file and outputs SRT.
// The outputBase should be the path without extension (e.g., /tmp/video123).
// OpenAI whisper names output after the input file: <output_dir>/<audio_stem>.srt
func (w *WhisperExecutor) Transcribe(ctx context.Context, audioPath string, outputBase string) (string, error) {
	outputDir := filepath.Dir(outputBase)

	_, err := w.commander.Execute(ctx, w.binary,
		audioPath,
		"--model", w.model,
		"--output_format", "srt",
		"--output_dir", outputDir,
		"--language", "en",
	)
	if err != nil {
		return "", fmt.Errorf("whisper transcription: %w", err)
	}

	// OpenAI whisper names output as <output_dir>/<input_stem>.srt
	audioStem := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	srtPath := filepath.Join(outputDir, audioStem+".srt")

	return srtPath, nil
}
