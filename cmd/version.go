package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the current version of the binary",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🖥  %s - version %s\n", name, Version)
	},
}
