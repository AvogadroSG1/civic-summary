package cmd

import (
	"fmt"

	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var quarantineCmd = &cobra.Command{
	Use:   "quarantine",
	Short: "Manage quarantined (failed) meetings",
	Long:  `View, retry, or remove quarantined meeting entries.`,
}

var quarantineListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List quarantined meetings",
	Example: `  civic-summary quarantine list --body=hagerstown`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		body, err := getBody(cmd, cfg)
		if err != nil {
			return err
		}

		quarantine := service.NewQuarantineService(cfg)
		entries, err := quarantine.ListQuarantined(body)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			output.Success("No quarantined items for %s", body.Name)
			return nil
		}

		output.Info("Quarantined items for %s:", body.Name)
		for _, e := range entries {
			fmt.Printf("  Video: %s\n", e.VideoID)
			fmt.Printf("    Date:    %s\n", e.MeetingDate)
			fmt.Printf("    Retries: %d\n", e.RetryCount)
			fmt.Printf("    Error:   %s\n", e.Error)
			fmt.Printf("    Since:   %s\n\n", e.QuarantinedAt.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

var quarantineRetryCmd = &cobra.Command{
	Use:   "retry [video-id]",
	Short: "Retry quarantined meetings",
	Long:  `Retries all quarantined meetings, or a specific one by video ID.`,
	Example: `  civic-summary quarantine retry --body=hagerstown
  civic-summary quarantine retry abc123 --body=hagerstown`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		body, err := getBody(cmd, cfg)
		if err != nil {
			return err
		}

		pipeline := buildPipeline(cfg)

		if len(args) > 0 {
			// Retry specific video.
			output.Info("Retrying video %s...", args[0])
			// The pipeline's retryQuarantined handles the logic.
			_, err := pipeline.ProcessBody(cmd.Context(), body, false)
			return err
		}

		// Retry all quarantined.
		_, err = pipeline.ProcessBody(cmd.Context(), body, false)
		return err
	},
}

var quarantineRemoveCmd = &cobra.Command{
	Use:     "remove <video-id>",
	Short:   "Remove a meeting from quarantine",
	Example: `  civic-summary quarantine remove abc123 --body=hagerstown`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		body, err := getBody(cmd, cfg)
		if err != nil {
			return err
		}

		quarantine := service.NewQuarantineService(cfg)
		if err := quarantine.Remove(body, args[0]); err != nil {
			return err
		}

		output.Success("Removed %s from quarantine", args[0])
		return nil
	},
}

func init() {
	quarantineCmd.PersistentFlags().String("body", "", "body slug")
	_ = quarantineCmd.MarkPersistentFlagRequired("body")

	quarantineCmd.AddCommand(quarantineListCmd)
	quarantineCmd.AddCommand(quarantineRetryCmd)
	quarantineCmd.AddCommand(quarantineRemoveCmd)
	rootCmd.AddCommand(quarantineCmd)
}
