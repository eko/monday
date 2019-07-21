package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/eko/monday/internal/config"
	"github.com/eko/monday/pkg/forwarder"
	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/runner"
	"github.com/eko/monday/pkg/watcher"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

const (
	name = "Monday"
)

var (
	Version string

	proxyComponent     *proxy.Proxy
	forwarderComponent *forwarder.Forwarder
	runnerComponent    *runner.Runner
	watcherComponent   *watcher.Watcher
)

func main() {
	rootCmd := &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			conf, err := config.Load()
			if err != nil {
				fmt.Printf("❌  %v", err)
				return
			}

			choice := selectProject(conf)
			run(conf, choice)

			handleExitSignal()
		},
	}

	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("❌  An error has occured during 'edit' command: %v\n", err)
		os.Exit(1)
	}
}

func selectProject(conf *config.Config) string {
	prompt := promptui.Select{
		Label: "Which project do you want to work on?",
		Items: conf.GetProjectNames(),
		Size:  20,
	}

	_, choice, err := prompt.Run()
	if err != nil {
		panic(fmt.Sprintf("Prompt failed:\n%v", err))
	}

	fmt.Print("\n")

	return choice
}

func run(conf *config.Config, choice string) {
	project, err := conf.GetProjectByName(choice)
	if err != nil {
		panic(err)
	}

	// Initializes hosts file manager
	hostfile, err := hostfile.NewClient()
	if err != nil {
		panic(err)
	}

	// Initializes proxy
	proxyComponent = proxy.NewProxy(hostfile)

	// Initializes runner
	runnerComponent = runner.NewRunner(proxyComponent, project)

	// Initializes forwarder
	forwarderComponent = forwarder.NewForwarder(proxyComponent, project)

	// Initializes watcher
	watcherComponent = watcher.NewWatcher(runnerComponent, forwarderComponent, conf.Watcher, project)
	watcherComponent.Watch()
}

// Handle for an exit signal in order to quit application on a proper way (shutting down connections and servers).
func handleExitSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	fmt.Println("\n👋  Bye, closing your local applications and remote connections now")

	forwarderComponent.Stop()
	proxyComponent.Stop()
	runnerComponent.Stop()
	watcherComponent.Stop()
}
