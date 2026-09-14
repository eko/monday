package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/eko/monday/internal/runtime"
	"github.com/eko/monday/pkg/build"
	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/forward"
	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/run"
	"github.com/eko/monday/pkg/setup"
	"github.com/eko/monday/pkg/ui"
	"github.com/eko/monday/pkg/watch"
	"github.com/eko/monday/pkg/write"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	name = "Monday"
)

var (
	Version string

	proxyfier proxy.Proxy
	forwarder forward.Forwarder
	setuper   setup.Setuper
	builder   build.Builder
	writer    write.Writer
	runner    run.Runner
	watcher   watch.Watcher

	uiEnabled bool
	stopOnce  sync.Once
)

// resolveUIEnabled returns whether the terminal UI must be enabled: it is the
// default in an interactive terminal, and can be disabled with the --no-ui
// flag or the MONDAY_NO_UI environment variable (e.g. for CI or piped output)
func resolveUIEnabled(cmd *cobra.Command) bool {
	if noUI, _ := cmd.Flags().GetBool("no-ui"); noUI {
		return false
	}

	if len(os.Getenv("MONDAY_NO_UI")) > 0 {
		return false
	}

	if forceUI, _ := cmd.Flags().GetBool("ui"); forceUI {
		return true
	}

	if len(os.Getenv("MONDAY_ENABLE_UI")) > 0 {
		return true
	}

	return term.IsTerminal(int(os.Stdout.Fd()))
}

func main() {
	ctx := context.Background()
	runtime.InitRuntimeEnvironment()

	rootCmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			uiEnabled = resolveUIEnabled(cmd)

			conf, err := config.Load()
			if err != nil {
				printConfigError(err)
				printValidateHint()
				os.Exit(1)
			}

			printBanner()

			runProject(ctx, conf, selectProject(conf))
		},
	}

	// Terminal UI flags (for both root and run commands)
	runCommand := runCmd(ctx)

	for _, command := range []*cobra.Command{rootCmd, runCommand} {
		command.Flags().Bool("no-ui", false, "Disable the terminal UI (plain text output)")
		command.Flags().Bool("ui", false, "Enable the terminal UI")

		if err := command.Flags().MarkDeprecated("ui", "the terminal UI is now enabled by default, use --no-ui to disable it"); err != nil {
			panic(err)
		}
	}

	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(runCommand)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌  An error has occured: %v\n", err)
		os.Exit(1)
	}
}

func selectProject(conf *config.Config) string {
	projects := conf.GetProjectNames()

	prompt := promptui.Select{
		Label: "Which project do you want to work on?",
		Items: projects,
		Size:  20,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }} (type to filter)",
			Active:   "▸ {{ . | cyan | bold }}",
			Inactive: "  {{ . }}",
			Selected: `⚡ Launching {{ . | cyan | bold }}...`,
		},
		Searcher: func(input string, index int) bool {
			return strings.Contains(
				strings.Replace(strings.ToLower(projects[index]), " ", "", -1),
				strings.Replace(strings.ToLower(input), " ", "", -1),
			)
		},
		StartInSearchMode: true,
	}

	_, choice, err := prompt.Run()
	if err != nil {
		if err.Error() == "^C" {
			fmt.Println("\n👋  Bye")
			os.Exit(0)
		}

		fmt.Printf("❌  An error has occured during the project selection: %v\n", err)
		os.Exit(1)
	}

	fmt.Print("\n")

	return choice
}

// runProject starts the given project and blocks until it is stopped
func runProject(ctx context.Context, conf *config.Config, choice string) {
	layout := ui.NewLayout(uiEnabled)
	layout.Init()
	layout.SetProject(choice)
	layout.SetVersion(Version)

	// Retrieve selected project configuration by its name
	project, err := conf.GetProjectByName(choice)
	if err != nil {
		fmt.Printf("❌  %v\n", err)
		fmt.Printf("Run '%s' to see the available projects\n", projectNameStyle.Render("monday list"))
		os.Exit(1)
	}

	// Prepend global configurations
	project.PrependApplications(conf.Applications)
	project.PrependForwards(conf.Forwards)

	// Initializes hosts file manager
	hostfile, err := hostfile.NewClient()
	if err != nil {
		fmt.Printf("❌  Unable to open the hosts file (try running monday with elevated privileges): %v\n", err)
		os.Exit(1)
	}

	proxyfier = proxy.NewProxy(layout.GetProxyView(), layout.GetProxyStatuses(), hostfile)
	setuper = setup.NewSetuper(layout.GetLogsView(), project, conf.Setup)
	builder = build.NewBuilder(layout.GetLogsView(), project, conf.Build)
	writer = write.NewWriter(layout.GetLogsView(), project)
	runner = run.NewRunner(layout.GetLogsView(), proxyfier, project, conf.Run)

	fwd := forward.NewForwarder(layout.GetForwardsView(), layout.GetForwardStatuses(), proxyfier, project)
	fwd.SetLogsView(layout.GetLogsView())
	forwarder = fwd

	layout.SetActions(fwd)

	watcher = watch.NewWatcher(setuper, builder, writer, runner, forwarder, conf.Watch, project)
	go watcher.Watch(ctx)

	waitForExit(ctx, layout)
}

// waitForExit blocks until the project must be stopped: the terminal UI is
// left, or an exit signal is received. Everything is then shut down properly
// (local applications, remote connections, hosts file entries, IP aliases).
func waitForExit(ctx context.Context, layout *ui.Layout) {
	exit := make(chan os.Signal, 1)

	if uiEnabled {
		// The terminal UI handles SIGINT and SIGTERM itself and restores the
		// terminal before returning: only a hangup (terminal closed) has to
		// be relayed so the shutdown still happens
		signal.Notify(exit, syscall.SIGHUP)

		go func() {
			<-exit
			layout.Quit()
		}()

		if err := layout.Run(); err != nil {
			fmt.Printf("❌  An error has occured while running the terminal UI: %v\n", err)
		}

		stopAll(ctx)

		return
	}

	signal.Notify(exit, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	<-exit

	stopAll(ctx)
}

// stopAll shuts down every component then exits, it runs at most once even
// when several exit conditions happen at the same time
func stopAll(ctx context.Context) {
	stopOnce.Do(func() {
		fmt.Println("\n👋  Bye, closing your local applications and remote connections now")

		// The shutdown must always complete: report an unexpected panic from one
		// of the components instead of crashing with a corrupted terminal
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("❌  An error has occured while shutting down: %v\n", r)
				os.Exit(1)
			}
		}()

		watcher.Stop()
		forwarder.Stop(ctx)
		proxyfier.Stop()
		runner.Stop()

		os.Exit(0)
	})
}

// printConfigError displays a configuration error, one problem per line
func printConfigError(err error) {
	lines := strings.Split(err.Error(), "\n")

	if len(lines) == 1 {
		fmt.Printf("❌  %s\n", lines[0])
		return
	}

	fmt.Println("❌  Invalid configuration:")

	for _, line := range lines {
		fmt.Printf("   • %s\n", line)
	}
}

// printValidateHint suggests the validate command after a configuration error
func printValidateHint() {
	fmt.Printf("\nRun '%s' to check your configuration files\n", projectNameStyle.Render("monday validate"))
}
