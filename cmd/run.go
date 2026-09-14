package main

import (
	"context"
	"os"

	"github.com/eko/monday/pkg/config"
	"github.com/spf13/cobra"
)

func runCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "This command allows you to run a specific project directly",
		Long: `In case you already have the project name you want to launch, you can launch it directly by using the run command
	and passing it as an argument`,
		Run: func(cmd *cobra.Command, args []string) {
			uiEnabled = resolveUIEnabled(cmd)

			conf, err := config.Load()
			if err != nil {
				printConfigError(err)
				printValidateHint()
				os.Exit(1)
			}

			printBanner()

			var choice string
			if len(args) > 0 {
				choice = args[0]
			} else {
				choice = selectProject(conf)
			}

			runProject(ctx, conf, choice)
			handleExitSignal(ctx)
		},
	}
}
