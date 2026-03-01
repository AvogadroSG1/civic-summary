package executor_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhisperExecutor_Transcribe(t *testing.T) {
	mock := executor.NewMockCommander()

	audioPath := "/tmp/test/video123.mp3"
	outputBase := "/tmp/test/video123"

	key := fmt.Sprintf("whisper %s --model medium --output_format srt --output_dir /tmp/test --language en", audioPath)
	mock.OnCommand(key, &executor.CommandResult{}, nil)

	w := executor.NewWhisperExecutor(mock, "whisper", "medium")
	srtPath, err := w.Transcribe(context.Background(), audioPath, outputBase)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/test/video123.srt", srtPath)
	assert.Len(t, mock.Calls, 1)
}

func TestWhisperExecutor_Transcribe_Error(t *testing.T) {
	mock := executor.NewMockCommander()

	audioPath := "/tmp/test/video123.mp3"
	outputBase := "/tmp/test/video123"

	key := fmt.Sprintf("whisper %s --model medium --output_format srt --output_dir /tmp/test --language en", audioPath)
	mock.OnCommand(key, nil, fmt.Errorf("whisper binary not found"))

	w := executor.NewWhisperExecutor(mock, "whisper", "medium")
	_, err := w.Transcribe(context.Background(), audioPath, outputBase)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "whisper transcription")
}
