package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show processing status for configured bodies",
	Example: `  civic-summary status
  civic-summary status --body=hagerstown`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		bodySlug, _ := cmd.Flags().GetString("body")

		bodies := cfg.Bodies
		if bodySlug != "" {
			body, err := cfg.GetBody(bodySlug)
			if err != nil {
				return err
			}
			bodies = map[string]domain.Body{bodySlug: body}
		}

		quarantine := service.NewQuarantineService(cfg)

		for slug := range bodies {
			body, _ := cfg.GetBody(slug)
			output.Banner(body.Name)

			// Count finalized summaries.
			finalizedDir := cfg.FinalizedDir(body)
			summaryCount := countSummaries(finalizedDir)
			fmt.Printf("  Finalized summaries: %d\n", summaryCount)

			// Count quarantined items.
			entries, _ := quarantine.ListQuarantined(body)
			fmt.Printf("  Quarantined:         %d\n", len(entries))
			for _, e := range entries {
				fmt.Printf("    - %s (date: %s, retries: %d, error: %s)\n",
					e.VideoID, e.MeetingDate, e.RetryCount, e.Error)
			}

			fmt.Println()
		}

		return nil
	},
}

func countSummaries(dir string) int {
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mdFiles, _ := filepath.Glob(filepath.Join(dir, entry.Name(), "*.md"))
		count += len(mdFiles)
	}
	return count
}

func init() {
	statusCmd.Flags().String("body", "", "body slug (default: all)")
	rootCmd.AddCommand(statusCmd)
}
