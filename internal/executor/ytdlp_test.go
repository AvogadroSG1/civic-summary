package executor_test

import (
	"context"
	"testing"

	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYtDlpExecutor_ListPlaylist(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "abc123|February 04, 2025 | Mayor & Council Regular Session\ndef456|January 21, 2025 | Mayor & Council Work Session\n",
	}

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	entries, err := ytdlp.ListPlaylist(context.Background(), "https://youtube.com/playlist?list=TEST", "2025")
	require.NoError(t, err)

	assert.Len(t, entries, 2)
	assert.Equal(t, "abc123", entries[0].VideoID)
	assert.Equal(t, "February 04, 2025 | Mayor & Council Regular Session", entries[0].Title)
	assert.Equal(t, "def456", entries[1].VideoID)
}

func TestYtDlpExecutor_ListPlaylist_EmptyOutput(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{Stdout: ""}

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	entries, err := ytdlp.ListPlaylist(context.Background(), "https://youtube.com/playlist?list=TEST", "")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestYtDlpExecutor_DownloadCaptions_NoCaptions(t *testing.T) {
	mock := executor.NewMockCommander()
	mock.DefaultResult = &executor.CommandResult{
		Stdout: "No subtitles available\n",
	}

	ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
	path, err := ytdlp.DownloadCaptions(context.Background(), "abc123", "/tmp")
	require.NoError(t, err)
	assert.Empty(t, path)
}

func TestParsePlaylistOutput(t *testing.T) {
	// Exported via the ListPlaylist method; test edge cases via mock.
	mock := executor.NewMockCommander()

	tests := []struct {
		name   string
		output string
		count  int
	}{
		{"normal", "id1|title1\nid2|title2\n", 2},
		{"trailing newlines", "id1|title1\n\n\n", 1},
		{"no pipe", "malformed\n", 0},
		{"empty", "", 0},
		{"pipe in title", "id1|title with | pipe\n", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.DefaultResult = &executor.CommandResult{Stdout: tt.output}
			ytdlp := executor.NewYtDlpExecutor(mock, "yt-dlp")
			entries, err := ytdlp.ListPlaylist(context.Background(), "url", "")
			require.NoError(t, err)
			assert.Len(t, entries, tt.count)
		})
	}
}
