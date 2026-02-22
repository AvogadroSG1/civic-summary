package cmd

import (
	"fmt"

	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/spf13/cobra"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Run the full processing pipeline",
	Long: `Runs all 5 pipeline stages (discovery, transcription, analysis,
cross-referencing, validation) for one or all configured bodies.

Without --body, processes all configured bodies sequentially.`,
	Example: `  civic-summary process --body=hagerstown
  civic-summary process --all
  civic-summary process --body=hagerstown --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		bodySlug, _ := cmd.Flags().GetString("body")
		all, _ := cmd.Flags().GetBool("all")

		pipeline := buildPipeline(cfg)

		if bodySlug != "" {
			body, err := cfg.GetBody(bodySlug)
			if err != nil {
				return err
			}
			stats, err := pipeline.ProcessBody(cmd.Context(), body, dryRun)
			if err != nil {
				return err
			}
			printStats(body.Name, stats.Discovered, stats.Processed, stats.Failed, stats.Quarantined)
			output.NotifyCompletion(body.Name, stats.Processed, stats.Failed, stats.Quarantined)
			return nil
		}

		if !all && len(cfg.Bodies) > 1 {
			return fmt.Errorf("multiple bodies configured; use --body=<slug> or --all")
		}

		allStats, err := pipeline.ProcessAll(cmd.Context(), dryRun)
		if err != nil {
			return err
		}

		for slug, stats := range allStats {
			printStats(slug, stats.Discovered, stats.Processed, stats.Failed, stats.Quarantined)
		}

		return nil
	},
}

func printStats(name string, discovered, processed, failed, quarantined int) {
	output.Banner(fmt.Sprintf("Summary: %s", name))
	fmt.Printf("  Discovered:  %d\n", discovered)
	fmt.Printf("  Processed:   %d\n", processed)
	fmt.Printf("  Failed:      %d\n", failed)
	fmt.Printf("  Quarantined: %d\n", quarantined)
}

func init() {
	processCmd.Flags().String("body", "", "body slug to process")
	processCmd.Flags().Bool("all", false, "process all configured bodies")
	processCmd.Flags().Bool("dry-run", false, "show what would be processed without executing")
	rootCmd.AddCommand(processCmd)
}
