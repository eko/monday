package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the current version of the binary",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(
			"%s %s\n",
			bannerBadgeStyle.Render("⚡ "+name),
			projectDetailStyle.Render("version "+Version),
		)
	},
}
