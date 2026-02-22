package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze <video-id>",
	Short: "Generate a summary for a specific video",
	Long: `Phase 3 only: sends the transcript to Claude CLI and generates
a citizen-friendly markdown summary.

Requires a transcript file to already exist in the output directory.`,
	Example: `  civic-summary analyze abc123 --body=hagerstown --date=2025-02-04
  civic-summary analyze xyz789 --body=bocc --date=2025-10-21 --transcript=/path/to/file.srt`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		videoID := args[0]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		body, err := getBody(cmd, cfg)
		if err != nil {
			return err
		}

		dateStr, _ := cmd.Flags().GetString("date")
		if dateStr == "" {
			return fmt.Errorf("--date flag is required (YYYY-MM-DD)")
		}
		meetingDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date format: %w", err)
		}

		meeting := domain.Meeting{
			VideoID:     videoID,
			MeetingDate: meetingDate,
			BodySlug:    body.Slug,
			MeetingType: "Regular Session",
		}

		// Load transcript.
		transcriptPath, _ := cmd.Flags().GetString("transcript")
		if transcriptPath == "" {
			transcriptPath = fmt.Sprintf("%s/%s/%s.srt",
				cfg.FinalizedDir(body), meeting.DateFolder(), videoID)
		}

		transcriptContent, err := os.ReadFile(transcriptPath)
		if err != nil {
			return fmt.Errorf("reading transcript: %w", err)
		}

		transcript := domain.Transcript{
			Content: string(transcriptContent),
			Path:    transcriptPath,
			Source:  domain.TranscriptSourceCaptions,
		}

		commander := executor.NewOsCommander()
		claude := executor.NewClaudeExecutor(commander, cfg.Tools.Claude)
		analysis := service.NewAnalysisService(claude, cfg.TemplateDir())

		summary, err := analysis.Analyze(cmd.Context(), meeting, transcript, body)
		if err != nil {
			return err
		}

		outputPath, _ := cmd.Flags().GetString("output")
		if outputPath == "" {
			fmt.Print(summary.Content)
		} else {
			if err := os.WriteFile(outputPath, []byte(summary.Content), 0o644); err != nil {
				return fmt.Errorf("writing summary: %w", err)
			}
			output.Success("Summary written to %s", outputPath)
		}

		return nil
	},
}

func init() {
	analyzeCmd.Flags().String("body", "", "body slug")
	analyzeCmd.Flags().String("date", "", "meeting date (YYYY-MM-DD)")
	analyzeCmd.Flags().String("transcript", "", "path to transcript file")
	analyzeCmd.Flags().String("output", "", "output file path (default: stdout)")
	_ = analyzeCmd.MarkFlagRequired("body")
	_ = analyzeCmd.MarkFlagRequired("date")
	rootCmd.AddCommand(analyzeCmd)
}
