package main

import (
	"fmt"
	"os/exec"

	"github.com/eko/monday/internal/config"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "This command allows you to open the configuration file in your default editor",
	Long: `For more information about the configuration, see the "example.yaml" file available
in the source code repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := config.CheckConfigFileExists()
		if err != nil {
			fmt.Printf("❌  %v\n", err)
			return
		}

		command := exec.Command("open", config.Filepath)

		if err := command.Start(); err != nil {
			fmt.Printf("❌  Cannot run the 'open' command to edit config file: %v\n", err)
			return
		}
	},
}
