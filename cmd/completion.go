package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use: "completion SHELL",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("requires 1 arg, found %d", len(args))
		}
		return cobra.OnlyValidArgs(cmd, args)
	},
	ValidArgs: []string{"bash", "zsh"},
	Short:     "Outputs shell completion for the given shell (bash or zsh)",
	Long:      "Outputs shell completion for the given shell (bash or zsh)",
	Run:       completion,
}

func completion(cmd *cobra.Command, args []string) {
	switch args[0] {
	case "bash":
		rootCmd(cmd).GenBashCompletion(os.Stdout)
	case "zsh":
		rootCmd(cmd).GenZshCompletion(os.Stdout)
	}
}

func rootCmd(cmd *cobra.Command) *cobra.Command {
	parent := cmd

	for parent.HasParent() {
		parent = parent.Parent()
	}

	return parent
}
