package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var crossrefCmd = &cobra.Command{
	Use:   "crossref <file>",
	Short: "Add Obsidian wikilinks to a summary file",
	Long: `Phase 4 only: scans a summary for date references and converts them
to Obsidian wikilinks when matching summaries exist.`,
	Example: `  civic-summary crossref summary.md --body=hagerstown --date=2025-02-04`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

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

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		meeting := domain.Meeting{
			MeetingDate: meetingDate,
			BodySlug:    body.Slug,
		}

		crossref := service.NewCrossReferenceService(cfg)
		result := crossref.AddCrossReferences(string(content), meeting, body)

		inPlace, _ := cmd.Flags().GetBool("in-place")
		if inPlace {
			if err := os.WriteFile(filePath, []byte(result), 0o644); err != nil {
				return fmt.Errorf("writing file: %w", err)
			}
			output.Success("Cross-references updated in %s", filePath)
		} else {
			fmt.Print(result)
		}

		return nil
	},
}

func init() {
	crossrefCmd.Flags().String("body", "", "body slug")
	crossrefCmd.Flags().String("date", "", "meeting date of the document (YYYY-MM-DD)")
	crossrefCmd.Flags().BoolP("in-place", "i", false, "modify file in place")
	_ = crossrefCmd.MarkFlagRequired("body")
	_ = crossrefCmd.MarkFlagRequired("date")
	rootCmd.AddCommand(crossrefCmd)
}
