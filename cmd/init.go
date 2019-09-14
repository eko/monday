package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/eko/monday/internal/runtime"
	"github.com/eko/monday/pkg/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "This command initializes a new configuration file and opens it in your favorite editor",
	Long: `For more information about the configuration, see the "example.yaml" file available
in the source code repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(config.Filepath); os.IsNotExist(err) {
			f, err := os.Create(config.Filepath)
			f.Close()
			if err != nil {
				fmt.Printf("❌  Cannot create config file in your home directory: %v\n", err)
				return
			}

			command := exec.Command(runtime.EditorCommand, config.Filepath)

			if err := command.Start(); err != nil {
				fmt.Printf("❌  Cannot run the '%s' command to edit config file: %v\n", runtime.EditorCommand, err)
				return
			}
		} else {
			fmt.Println("❌  You already have a configuration file. Please use 'edit' command")
			return
		}
	},
}
