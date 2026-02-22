package executor

import (
	"context"
	"fmt"
	"strings"
)

// PlaylistEntry represents a single video discovered from a YouTube playlist.
type PlaylistEntry struct {
	VideoID string
	Title   string
}

// YtDlpExecutor wraps yt-dlp operations.
type YtDlpExecutor struct {
	commander Commander
	binary    string
}

// NewYtDlpExecutor creates a new YtDlpExecutor.
func NewYtDlpExecutor(commander Commander, binary string) *YtDlpExecutor {
	return &YtDlpExecutor{
		commander: commander,
		binary:    binary,
	}
}

// ListPlaylist returns all videos in a playlist, optionally filtered by year.
func (y *YtDlpExecutor) ListPlaylist(ctx context.Context, playlistURL string, yearFilter string) ([]PlaylistEntry, error) {
	args := []string{
		"--flat-playlist",
		"--print", "%(id)s|%(title)s",
	}
	if yearFilter != "" {
		args = append(args, "--match-filter", fmt.Sprintf("title ~= '%s'", yearFilter))
	}
	args = append(args, playlistURL)

	result, err := y.commander.Execute(ctx, y.binary, args...)
	if err != nil {
		return nil, fmt.Errorf("listing playlist: %w", err)
	}

	return parsePlaylistOutput(result.Stdout)
}

// DownloadCaptions downloads auto-generated captions in SRT format.
// Returns the path to the downloaded SRT file, or empty string if unavailable.
func (y *YtDlpExecutor) DownloadCaptions(ctx context.Context, videoID string, outputDir string) (string, error) {
	videoURL := "https://www.youtube.com/watch?v=" + videoID

	// Check if auto-captions are available.
	checkResult, err := y.commander.Execute(ctx, y.binary, "--list-subs", videoURL)
	if err != nil {
		return "", fmt.Errorf("checking captions: %w", err)
	}

	hasCaptions := strings.Contains(checkResult.Stdout, "Available automatic captions") ||
		strings.Contains(checkResult.Stderr, "Available automatic captions")
	if !hasCaptions {
		return "", nil
	}

	// Download captions.
	outputTemplate := fmt.Sprintf("%s/%s", outputDir, videoID)
	_, err = y.commander.Execute(ctx, y.binary,
		"--write-auto-subs",
		"--sub-lang", "en",
		"--sub-format", "srt",
		"--skip-download",
		"--output", outputTemplate,
		videoURL,
	)
	if err != nil {
		return "", fmt.Errorf("downloading captions: %w", err)
	}

	return fmt.Sprintf("%s/%s.en.srt", outputDir, videoID), nil
}

// DownloadAudio downloads audio in MP3 format for Whisper fallback.
func (y *YtDlpExecutor) DownloadAudio(ctx context.Context, videoID string, outputPath string) error {
	videoURL := "https://www.youtube.com/watch?v=" + videoID

	_, err := y.commander.Execute(ctx, y.binary,
		"--extract-audio",
		"--audio-format", "mp3",
		"--output", outputPath,
		videoURL,
	)
	if err != nil {
		return fmt.Errorf("downloading audio: %w", err)
	}
	return nil
}

// GetDescription downloads the video description.
func (y *YtDlpExecutor) GetDescription(ctx context.Context, videoID string) (string, error) {
	videoURL := "https://www.youtube.com/watch?v=" + videoID

	result, err := y.commander.Execute(ctx, y.binary,
		"--skip-download",
		"--print", "%(description)s",
		videoURL,
	)
	if err != nil {
		return "", fmt.Errorf("getting description: %w", err)
	}
	return strings.TrimSpace(result.Stdout), nil
}

// parsePlaylistOutput parses the "ID|Title" format output from yt-dlp.
func parsePlaylistOutput(output string) ([]PlaylistEntry, error) {
	var entries []PlaylistEntry
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, PlaylistEntry{
			VideoID: strings.TrimSpace(parts[0]),
			Title:   strings.TrimSpace(parts[1]),
		})
	}
	return entries, nil
}
