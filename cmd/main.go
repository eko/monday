package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/forwarder"
	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/runner"
	"github.com/eko/monday/pkg/ui"
	"github.com/eko/monday/pkg/watcher"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jroimartin/gocui"
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

	rootCmd.AddCommand(completionCmd)
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
	layout := ui.NewLayout()
	layout.Init()
	defer layout.GetGui().Close()

	if err := layout.GetGui().SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		panic(err)
	}

	layout.GetStatusView().Writef(" ⇢  %s | Commands: ←/→: select view | ↑/↓: scroll up/down | a: toggle autoscroll | f: toggle fullscreen", choice)

	project, err := conf.GetProjectByName(choice)
	if err != nil {
		panic(err)
	}

	// Initializes hosts file manager
	hostfile, err := hostfile.NewClient()
	if err != nil {
		panic(err)
	}

	proxyComponent = proxy.NewProxy(layout.GetProxyView(), hostfile)
	runnerComponent = runner.NewRunner(layout.GetLogsView(), proxyComponent, project)
	forwarderComponent = forwarder.NewForwarder(layout.GetForwardsView(), proxyComponent, project)

	watcherComponent = watcher.NewWatcher(runnerComponent, forwarderComponent, conf.Watcher, project)
	watcherComponent.Watch()

	if err := layout.GetGui().MainLoop(); err != nil && err != gocui.ErrQuit {
		fmt.Println(err)
		stopAll()
	}
}

// Handle for an exit signal in order to quit application on a proper way (shutting down connections and servers).
func handleExitSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	stopAll()
}

func stopAll() {
	fmt.Println("\n👋  Bye, closing your local applications and remote connections now")

	forwarderComponent.Stop()
	proxyComponent.Stop()
	runnerComponent.Stop()
	watcherComponent.Stop()

	os.Exit(0)
}

func quit(g *gocui.Gui, v *gocui.View) error {
	g.Close()
	stopAll()
	return gocui.ErrQuit
}
