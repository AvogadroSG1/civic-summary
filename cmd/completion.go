package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for civic-summary.

To load completions:

Bash:
  $ source <(civic-summary completion bash)
  # To load completions for each session:
  $ civic-summary completion bash > /etc/bash_completion.d/civic-summary

Zsh:
  $ source <(civic-summary completion zsh)
  # To load completions for each session:
  $ civic-summary completion zsh > "${fpath[1]}/_civic-summary"

Fish:
  $ civic-summary completion fish | source
  # To load completions for each session:
  $ civic-summary completion fish > ~/.config/fish/completions/civic-summary.fish`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
