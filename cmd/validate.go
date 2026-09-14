package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/eko/monday/pkg/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration files without running anything",
	Long: `Loads the configuration files, reports the unknown fields (with their file and line)
and checks that every project, local application and forward holds the values
needed to run it.`,
	Run: func(cmd *cobra.Command, args []string) {
		loaded, err := config.LoadDetailed()

		if loaded != nil && len(loaded.Files) > 0 {
			fmt.Println("📄  Configuration files:")

			for _, file := range loaded.Files {
				fmt.Printf("   • %s\n", file)
			}

			fmt.Println()
		}

		if err != nil {
			printConfigError(err)
			os.Exit(1)
		}

		applications, forwards := countEntries(loaded.Config)

		fmt.Printf(
			"✅  Configuration is valid: %d project(s), %d local application(s), %d forward(s)\n",
			len(loaded.Config.Projects),
			applications,
			forwards,
		)

		if len(loaded.IgnoredKeys) > 0 {
			fmt.Printf(
				"ℹ️   Ignored top-level keys (only YAML anchors are expected there): %s\n",
				strings.Join(loaded.IgnoredKeys, ", "),
			)
		}
	},
}

// countEntries returns the number of local applications and forwards declared
// globally and in every project
func countEntries(conf *config.Config) (int, int) {
	applications := len(conf.Applications)
	forwards := len(conf.Forwards)

	for _, project := range conf.Projects {
		applications += len(project.Applications)
		forwards += len(project.Forwards)
	}

	return applications, forwards
}
