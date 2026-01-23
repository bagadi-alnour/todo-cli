package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate autocompletion scripts for your shell.

To enable completions:

  Bash:
    source <(todo completion bash)
    # or add above line to ~/.bashrc

  Zsh:
    todo completion zsh > "${fpath[1]}/_todo"
    autoload -U compinit && compinit

  Fish:
    todo completion fish | source
    # or save to ~/.config/fish/completions/todo.fish

  PowerShell:
    todo completion powershell | Out-String | Invoke-Expression
    # or save to $PROFILE`,
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := args[0]
		var err error
		switch sh {
		case "bash":
			err = rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			err = rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			err = rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			err = fmt.Errorf("unsupported shell: %s", sh)
		}
		return err
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
