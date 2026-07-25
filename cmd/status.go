package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AvogadroSG1/civic-summary/internal/domain"
	"github.com/AvogadroSG1/civic-summary/internal/llm"
	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show processing status for configured bodies",
	Long: `Reports finalized and quarantined summary counts per body, and checks
that the configured language model is reachable.

The model check sends one minimal request per distinct provider and model, which
catches a bad API key or model name before a run wastes a transcription. Pass
--skip-llm-check to stay offline.`,
	Example: `  civic-summary status
  civic-summary status --body=hagerstown
  civic-summary status --skip-llm-check`,
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
		skipLLM, _ := cmd.Flags().GetBool("skip-llm-check")

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

			reportLLM(cmd.Context(), cfg.ResolveLLM(body), skipLLM)

			fmt.Println()
		}

		return nil
	},
}

// reportLLM prints a body's resolved model configuration and, unless skipped,
// the result of a live reachability probe.
func reportLLM(ctx context.Context, llmCfg domain.LLMConfig, skip bool) {
	fmt.Printf("  Model:               %s\n", llmCfg.Describe())
	if llmCfg.BaseURL != "" {
		fmt.Printf("  Endpoint:            %s\n", llmCfg.BaseURL)
	}

	if skip {
		fmt.Printf("  Model check:         skipped\n")
		return
	}

	client, err := llm.New(llmCfg)
	if err != nil {
		output.Failure("Model check: %v", err)
		return
	}
	if err := client.Ping(ctx); err != nil {
		output.Failure("Model check: %v", err)
		return
	}

	output.Success("Model check: reachable")
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
	statusCmd.Flags().Bool("skip-llm-check", false, "skip the language model reachability probe")
	rootCmd.AddCommand(statusCmd)
}
