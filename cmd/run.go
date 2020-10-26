package main

import (
	"fmt"
	"strconv"

	"github.com/eko/monday/pkg/config"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "This command allows you to run a specific project directly",
	Long: `In case you already have the project name you want to launch, you can launch it directly by using the run command
and passing it as an argument`,
	Run: func(cmd *cobra.Command, args []string) {
		if !uiEnabled {
			uiEnabled, _ = strconv.ParseBool(cmd.Flag("ui").Value.String())
		}

		conf, err := config.Load()
		if err != nil {
			fmt.Printf("❌  %v\n", err)
			return
		}

		var choice string
		if len(args) > 0 {
			choice = args[0]
		} else {
			choice = selectProject(conf)
		}

		runProject(conf, choice)
		handleExitSignal()
	},
}
