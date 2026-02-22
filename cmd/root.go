// Package cmd implements the Cobra command tree for civic-summary.
package cmd

import (
	"os"

	"github.com/AvogadroSG1/civic-summary/internal/output"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "civic-summary",
	Short: "Process government meeting videos into citizen-friendly summaries",
	Long: `civic-summary transforms YouTube recordings of government meetings
into accessible, well-structured Obsidian markdown summaries.

It supports multiple government bodies via configuration and handles
the complete pipeline: discovery, transcription, analysis, cross-referencing,
and validation.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		output.SetupLogging(verbose)
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ~/.civic-summary/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
