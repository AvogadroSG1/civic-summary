package service_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranscriptionService_ValidateTranscript(t *testing.T) {
	mock := executor.NewMockCommander()
	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	svc := service.NewTranscriptionService(ytdlp, nil)

	tests := []struct {
		name       string
		transcript domain.Transcript
		wantErr    bool
	}{
		{
			"empty transcript",
			domain.Transcript{Content: ""},
			true,
		},
		{
			"too short",
			domain.Transcript{Content: "just a few words here"},
			true,
		},
		{
			"valid transcript",
			domain.Transcript{Content: generateWords(600)},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateTranscript(tt.transcript)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func generateWords(n int) string {
	result := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			result += " "
		}
		result += "word"
	}
	return result
}

// transcribeMeeting returns a deterministic meeting for transcription tests.
func transcribeMeeting() domain.Meeting {
	return domain.Meeting{
		VideoID:     "test123",
		Title:       "February 04, 2025 | Council Session",
		MeetingDate: time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
		MeetingType: "Regular Session",
		BodySlug:    "hagerstown",
	}
}

func TestTranscriptionService_Transcribe_CaptionsSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	meeting := transcribeMeeting()
	videoURL := "https://www.youtube.com/watch?v=" + meeting.VideoID

	mock := executor.NewMockCommander()

	// Step 1: --list-subs check succeeds — captions are available.
	listSubsKey := fmt.Sprintf("yt-dlp --list-subs %s", videoURL)
	mock.OnCommand(listSubsKey, &executor.CommandResult{
		Stdout: "Available automatic captions for test123:\nen  English",
	}, nil)

	// Step 2: Download captions succeeds.
	outputTmpl := fmt.Sprintf("%s/%s", tmpDir, meeting.VideoID)
	downloadKey := fmt.Sprintf("yt-dlp --write-auto-subs --sub-lang en --sub-format srt --skip-download --output %s %s", outputTmpl, videoURL)
	mock.OnCommand(downloadKey, &executor.CommandResult{}, nil)

	// Pre-create the SRT file that yt-dlp would produce.
	srtPath := filepath.Join(tmpDir, meeting.VideoID+".en.srt")
	srtContent := "1\n00:00:01,000 --> 00:00:05,000\nThe meeting will come to order.\n"
	require.NoError(t, os.WriteFile(srtPath, []byte(srtContent), 0o644))

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	svc := service.NewTranscriptionService(ytdlp, nil)

	transcript, err := svc.Transcribe(context.Background(), meeting, tmpDir)
	require.NoError(t, err)

	assert.Equal(t, srtContent, transcript.Content)
	assert.Equal(t, domain.TranscriptSourceCaptions, transcript.Source)
	// Should have been renamed from .en.srt to .srt.
	assert.Equal(t, filepath.Join(tmpDir, meeting.VideoID+".srt"), transcript.Path)
}

func TestTranscriptionService_Transcribe_CaptionsFail_WhisperFallback(t *testing.T) {
	tmpDir := t.TempDir()
	meeting := transcribeMeeting()
	videoURL := "https://www.youtube.com/watch?v=" + meeting.VideoID

	mock := executor.NewMockCommander()

	// Captions check returns no available captions.
	listSubsKey := fmt.Sprintf("yt-dlp --list-subs %s", videoURL)
	mock.OnCommand(listSubsKey, &executor.CommandResult{
		Stdout: "No subtitles available for test123",
	}, nil)

	// Whisper: audio download succeeds.
	audioPath := filepath.Join(tmpDir, meeting.VideoID+".mp3")
	downloadAudioKey := fmt.Sprintf("yt-dlp --extract-audio --audio-format mp3 --output %s %s", audioPath, videoURL)
	mock.OnCommand(downloadAudioKey, &executor.CommandResult{}, nil)

	// Whisper: transcription succeeds.
	outputBase := filepath.Join(tmpDir, meeting.VideoID)
	whisperKey := fmt.Sprintf("whisper-cli -m model.bin -f %s --output-srt --output-file %s --language en", audioPath, outputBase)
	mock.OnCommand(whisperKey, &executor.CommandResult{}, nil)

	// Pre-create the SRT file that whisper would produce.
	whisperSrtPath := outputBase + ".srt"
	srtContent := "1\n00:00:01,000 --> 00:00:05,000\nWhisper transcription output.\n"
	require.NoError(t, os.WriteFile(whisperSrtPath, []byte(srtContent), 0o644))

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	whisper := executor.NewWhisperExecutor(mock, "whisper-cli", "model.bin")
	svc := service.NewTranscriptionService(ytdlp, whisper)

	transcript, err := svc.Transcribe(context.Background(), meeting, tmpDir)
	require.NoError(t, err)

	assert.Equal(t, "1\n00:00:01,000 --> 00:00:05,000\nWhisper transcription output.", transcript.Content)
	assert.Equal(t, domain.TranscriptSourceWhisper, transcript.Source)
	assert.Equal(t, whisperSrtPath, transcript.Path)
}

func TestTranscriptionService_Transcribe_WhisperNotConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	meeting := transcribeMeeting()
	videoURL := "https://www.youtube.com/watch?v=" + meeting.VideoID

	mock := executor.NewMockCommander()

	// Captions check fails — no captions available.
	listSubsKey := fmt.Sprintf("yt-dlp --list-subs %s", videoURL)
	mock.OnCommand(listSubsKey, &executor.CommandResult{
		Stdout: "No subtitles",
	}, nil)

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	// Whisper is nil — not configured.
	svc := service.NewTranscriptionService(ytdlp, nil)

	_, err := svc.Transcribe(context.Background(), meeting, tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "whisper not configured")
}

func TestTranscriptionService_Transcribe_CaptionsRenamesSrt(t *testing.T) {
	tmpDir := t.TempDir()
	meeting := transcribeMeeting()
	videoURL := "https://www.youtube.com/watch?v=" + meeting.VideoID

	mock := executor.NewMockCommander()

	listSubsKey := fmt.Sprintf("yt-dlp --list-subs %s", videoURL)
	mock.OnCommand(listSubsKey, &executor.CommandResult{
		Stdout: "Available automatic captions for test123:\nen  English",
	}, nil)

	outputTmpl := fmt.Sprintf("%s/%s", tmpDir, meeting.VideoID)
	downloadKey := fmt.Sprintf("yt-dlp --write-auto-subs --sub-lang en --sub-format srt --skip-download --output %s %s", outputTmpl, videoURL)
	mock.OnCommand(downloadKey, &executor.CommandResult{}, nil)

	// Create .en.srt file.
	enSrtPath := filepath.Join(tmpDir, meeting.VideoID+".en.srt")
	require.NoError(t, os.WriteFile(enSrtPath, []byte("subtitle content"), 0o644))

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	svc := service.NewTranscriptionService(ytdlp, nil)

	transcript, err := svc.Transcribe(context.Background(), meeting, tmpDir)
	require.NoError(t, err)

	// Verify rename happened: .en.srt should no longer exist.
	_, err = os.Stat(enSrtPath)
	assert.True(t, os.IsNotExist(err), ".en.srt should have been renamed")

	// .srt should exist.
	finalPath := filepath.Join(tmpDir, meeting.VideoID+".srt")
	assert.Equal(t, finalPath, transcript.Path)
	_, err = os.Stat(finalPath)
	assert.NoError(t, err, ".srt should exist after rename")
}
