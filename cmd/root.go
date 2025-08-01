package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cmcp",
	Short: "A CLI tool to manage MCP servers",
	Long:  `cmcp is a command-line tool for managing Model Context Protocol (MCP) servers on your system.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(onlineCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:
  $ source <(cmcp completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ cmcp completion bash > /etc/bash_completion.d/cmcp
  # macOS:
  $ cmcp completion bash > $(brew --prefix)/etc/bash_completion.d/cmcp

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ cmcp completion zsh > "${fpath[1]}/_cmcp"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ cmcp completion fish | source

  # To load completions for each session, execute once:
  $ cmcp completion fish > ~/.config/fish/completions/cmcp.fish

PowerShell:
  PS> cmcp completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> cmcp completion powershell > cmcp.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
