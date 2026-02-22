package executor

import (
	"context"
	"fmt"
)

// WhisperExecutor wraps whisper.cpp operations for audio transcription.
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

// Transcribe runs whisper.cpp on an audio file and outputs SRT.
// The outputBase should be the path without extension; whisper appends ".srt".
func (w *WhisperExecutor) Transcribe(ctx context.Context, audioPath string, outputBase string) (string, error) {
	_, err := w.commander.Execute(ctx, w.binary,
		"-m", w.model,
		"-f", audioPath,
		"--output-srt",
		"--output-file", outputBase,
		"--language", "en",
	)
	if err != nil {
		return "", fmt.Errorf("whisper transcription: %w", err)
	}

	return outputBase + ".srt", nil
}
