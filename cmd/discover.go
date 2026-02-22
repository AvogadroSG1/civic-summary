package cmd

import (
	"fmt"

	"github.com/AvogadroSG1/civic-summary/internal/executor"
	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Find unprocessed videos from a body's playlist",
	Long:  `Phase 1 only: lists videos in the playlist that don't have finalized summaries.`,
	Example: `  civic-summary discover --body=hagerstown
  civic-summary discover --body=bocc`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		body, err := getBody(cmd, cfg)
		if err != nil {
			return err
		}

		commander := executor.NewOsCommander()
		ytdlp := executor.NewYtDlpExecutor(commander, cfg.Tools.YtDlp)
		discovery := service.NewDiscoveryService(ytdlp, cfg)

		meetings, err := discovery.DiscoverNewMeetings(cmd.Context(), body)
		if err != nil {
			return err
		}

		if len(meetings) == 0 {
			output.Success("No new videos for %s", body.Name)
			return nil
		}

		output.Info("Found %d new video(s) for %s:", len(meetings), body.Name)
		for _, m := range meetings {
			fmt.Printf("  %s | %s | %s\n", m.VideoID, m.ISODate(), m.Title)
		}

		return nil
	},
}

func init() {
	discoverCmd.Flags().String("body", "", "body slug to discover")
	_ = discoverCmd.MarkFlagRequired("body")
	rootCmd.AddCommand(discoverCmd)
}
