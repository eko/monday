package main

import (
	"fmt"

	"github.com/eko/monday/pkg/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects defined in your configuration",
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.Load()
		if err != nil {
			fmt.Printf("❌  %v\n", err)
			return
		}

		printBanner()

		for _, project := range conf.Projects {
			details := fmt.Sprintf(
				"%d application(s) · %d forward(s)",
				len(project.Applications)+len(conf.Applications),
				len(project.Forwards)+len(conf.Forwards),
			)

			fmt.Printf(
				"  %s  %s\n",
				projectNameStyle.Render(project.Name),
				projectDetailStyle.Render(details),
			)
		}

		fmt.Println()
		fmt.Printf("Run a project with: %s\n", projectNameStyle.Render("monday run <project>"))
	},
}
