// Package output provides terminal formatting, logging setup, and macOS notifications.
package output

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// SetupLogging configures slog for structured logging to stdout (12-Factor XI).
func SetupLogging(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

// Banner prints a formatted section header.
func Banner(title string) {
	line := strings.Repeat("=", 55)
	fmt.Println(line)
	fmt.Printf("  %s\n", title)
	fmt.Println(line)
}

// Status prints a status line with an icon prefix.
func Status(icon string, format string, args ...interface{}) {
	fmt.Printf("%s %s\n", icon, fmt.Sprintf(format, args...))
}

// Success prints a success message.
func Success(format string, args ...interface{}) {
	Status("\u2713", format, args...)
}

// Failure prints a failure message.
func Failure(format string, args ...interface{}) {
	Status("\u2717", format, args...)
}

// Warning prints a warning message.
func Warning(format string, args ...interface{}) {
	Status("\u26a0", format, args...)
}

// Info prints an info message.
func Info(format string, args ...interface{}) {
	Status("\u2139", format, args...)
}

// NotifyMacOS sends a macOS notification via osascript.
func NotifyMacOS(title, message, sound string) {
	if sound == "" {
		sound = "default"
	}
	script := fmt.Sprintf(
		`display notification %q with title %q sound name %q`,
		message, title, sound,
	)
	// Best-effort notification; ignore errors (headless environments).
	_ = exec.Command("osascript", "-e", script).Run()
}

// NotifyCompletion sends a summary notification based on processing results.
func NotifyCompletion(appName string, processed, failed, quarantined int) {
	switch {
	case failed > 0:
		NotifyMacOS(appName,
			fmt.Sprintf("%d processed, %d failed", processed, failed),
			"Basso")
	case quarantined > 0:
		NotifyMacOS(appName,
			fmt.Sprintf("%d processed, %d in quarantine", processed, quarantined),
			"default")
	default:
		NotifyMacOS(appName,
			fmt.Sprintf("Processed %d videos successfully", processed),
			"default")
	}
}
