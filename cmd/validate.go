package cmd

import (
	"fmt"
	"os"

	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/AvogadroSG1/civic-summary/internal/service"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a summary file against quality requirements",
	Long: `Phase 5 only: checks frontmatter, required sections, word count,
timestamps, and Claude meta-commentary.`,
	Example: `  civic-summary validate summary.md --body=hagerstown
  civic-summary validate *.md --body=bocc`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		body, err := getBody(cmd, cfg)
		if err != nil {
			return err
		}

		validation := service.NewValidationService()
		hasErrors := false

		for _, filePath := range args {
			content, err := os.ReadFile(filePath)
			if err != nil {
				output.Failure("Cannot read %s: %s", filePath, err)
				hasErrors = true
				continue
			}

			result := validation.Validate(string(content), body)

			fmt.Printf("\n--- %s ---\n", filePath)
			if result.IsValid() {
				output.Success("PASS")
			} else {
				output.Failure("FAIL")
				hasErrors = true
			}

			for _, issue := range result.Issues {
				fmt.Printf("  %s\n", issue)
			}
		}

		if hasErrors {
			return fmt.Errorf("validation failed")
		}

		return nil
	},
}

func init() {
	validateCmd.Flags().String("body", "", "body slug for body-specific validation rules")
	_ = validateCmd.MarkFlagRequired("body")
	rootCmd.AddCommand(validateCmd)
}
