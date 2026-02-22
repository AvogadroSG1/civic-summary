package cmd

import (
	"fmt"
	"strings"

	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/spf13/cobra"
)

var bodiesCmd = &cobra.Command{
	Use:   "bodies",
	Short: "View configured government bodies",
}

var bodiesListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all configured bodies",
	Example: `  civic-summary bodies list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		output.Info("Configured bodies:")
		for slug, body := range cfg.Bodies {
			fmt.Printf("  %s: %s\n", slug, body.Name)
			fmt.Printf("    Source: %s\n", body.DiscoveryURL())
			fmt.Printf("    Template: %s\n", body.PromptTemplate)
			fmt.Println()
		}

		return nil
	},
}

var bodiesShowCmd = &cobra.Command{
	Use:     "show <slug>",
	Short:   "Show detailed configuration for a body",
	Example: `  civic-summary bodies show hagerstown`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		body, err := cfg.GetBody(args[0])
		if err != nil {
			return err
		}

		output.Banner(body.Name)
		fmt.Printf("  Slug:             %s\n", body.Slug)
		if body.PlaylistID != "" {
			fmt.Printf("  Playlist ID:      %s\n", body.PlaylistID)
		}
		if body.VideoSourceURL != "" {
			fmt.Printf("  Video Source URL:  %s\n", body.VideoSourceURL)
		}
		fmt.Printf("  Discovery URL:    %s\n", body.DiscoveryURL())
		fmt.Printf("  Output Subdir:    %s\n", body.OutputSubdir)
		fmt.Printf("  Filename Pattern: %s\n", body.FilenamePattern)
		fmt.Printf("  Date Regex:       %s\n", body.TitleDateRegex)
		fmt.Printf("  Prompt Template:  %s\n", body.PromptTemplate)
		fmt.Printf("  Author:           %s\n", body.Author)
		fmt.Printf("  Tags:             %s\n", strings.Join(body.Tags, ", "))
		if len(body.MeetingTypes) > 0 {
			fmt.Printf("  Meeting Types:    %s\n", strings.Join(body.MeetingTypes, ", "))
		}
		fmt.Printf("  Output Dir:       %s\n", cfg.BodyOutputDir(body))
		fmt.Printf("  Finalized Dir:    %s\n", cfg.FinalizedDir(body))

		return nil
	},
}

func init() {
	bodiesCmd.AddCommand(bodiesListCmd)
	bodiesCmd.AddCommand(bodiesShowCmd)
	rootCmd.AddCommand(bodiesCmd)
}
