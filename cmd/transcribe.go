package cmd

import (
	"fmt"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var transcribeCmd = &cobra.Command{
	Use:   "transcribe <video-id>",
	Short: "Get transcript for a specific video",
	Long:  `Phase 2 only: downloads captions or runs Whisper to produce an SRT transcript.`,
	Example: `  civic-summary transcribe abc123 --body=hagerstown
  civic-summary transcribe xyz789 --body=bocc --output-dir=/tmp`,
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

		outputDir, _ := cmd.Flags().GetString("output-dir")
		if outputDir == "" {
			outputDir = cfg.FinalizedDir(body)
		}

		commander := executor.NewOsCommander()
		ytdlp := executor.NewYtDlpExecutor(commander, cfg.Tools.YtDlp)
		var whisper *executor.WhisperExecutor
		if cfg.Tools.Whisper != "" {
			whisper = executor.NewWhisperExecutor(commander, cfg.Tools.Whisper, cfg.Tools.WhisperModel)
		}
		transcription := service.NewTranscriptionService(ytdlp, whisper)

		meeting := domain.Meeting{
			VideoID:  videoID,
			BodySlug: body.Slug,
		}

		transcript, err := transcription.Transcribe(cmd.Context(), meeting, outputDir)
		if err != nil {
			return err
		}

		if err := transcription.ValidateTranscript(transcript); err != nil {
			return err
		}

		output.Success("Transcript saved: %s (%d words, source: %s)",
			transcript.Path, transcript.WordCount(), transcript.Source)
		fmt.Println(transcript.Path)

		return nil
	},
}

func init() {
	transcribeCmd.Flags().String("body", "", "body slug")
	transcribeCmd.Flags().String("output-dir", "", "output directory (default: body's finalized dir)")
	_ = transcribeCmd.MarkFlagRequired("body")
	rootCmd.AddCommand(transcribeCmd)
}
