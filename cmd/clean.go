package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/hostfile"
	"github.com/eko/monday/pkg/proxy"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove the hosts file entries and loopback IP aliases left by a previous run",
	Long: `Monday cleans up after itself when it stops properly, but a crash or a forced kill
can leave hostnames in your hosts file and IP aliases on the loopback interface.

This command removes the hostnames declared in your configuration from the hosts
file, and the 127.0.1.0+ addresses assigned to the loopback interface. Like running
a project, it needs elevated privileges.`,
	Run: func(cmd *cobra.Command, args []string) {
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		if err := clean(dryRun); err != nil {
			fmt.Printf("❌  %v\n", err)
			os.Exit(1)
		}
	},
}

func clean(dryRun bool) error {
	verb := "Removed"
	if dryRun {
		verb = "Would remove"
	}

	removed, failed := 0, 0

	// Hosts file entries
	if hosts, hostnames, err := configuredHosts(); err != nil {
		fmt.Printf("⚠️   %v\n", err)
	} else {
		for _, hostname := range hostnames {
			if !hosts.HasHost(hostname) {
				continue
			}

			if !dryRun {
				if err := hosts.RemoveHost(hostname); err != nil {
					fmt.Printf("❌  Unable to remove '%s' from the hosts file: %v\n", hostname, err)
					failed++

					continue
				}
			}

			fmt.Printf("🧹  %s '%s' from the hosts file\n", verb, hostname)
			removed++
		}
	}

	// Loopback IP aliases
	aliases, err := proxy.ListLoopbackAliases()
	if err != nil {
		fmt.Printf("❌  Unable to list the loopback interface addresses: %v\n", err)
		failed++
	}

	for _, ip := range aliases {
		if !dryRun {
			if err := proxy.RemoveLoopbackAlias(ip); err != nil {
				if errors.Is(err, proxy.ErrAliasRemovalUnsupported) {
					fmt.Printf("⚠️   %v, the IP aliases are left as is\n", err)
					break
				}

				fmt.Printf("❌  Unable to remove IP address '%s' from the loopback interface: %v\n", ip, err)
				failed++

				continue
			}
		}

		fmt.Printf("🧹  %s IP address '%s' from the loopback interface\n", verb, ip)
		removed++
	}

	if failed > 0 {
		return fmt.Errorf("%d item(s) could not be cleaned, try running with elevated privileges: sudo monday clean", failed)
	}

	switch {
	case removed == 0:
		fmt.Println("✅  Nothing to clean, your system is already tidy")
	case dryRun:
		fmt.Printf("ℹ️   %d item(s) would be cleaned, run without --dry-run to remove them\n", removed)
	default:
		fmt.Printf("✅  %d item(s) cleaned\n", removed)
	}

	return nil
}

// configuredHosts opens the hosts file and returns every hostname the
// configuration maps locally
func configuredHosts() (hostfile.Hostfile, []string, error) {
	conf, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf(
			"unable to load the configuration, the hosts file entries are skipped: %s",
			strings.Split(err.Error(), "\n")[0],
		)
	}

	hosts, err := hostfile.NewClient()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open the hosts file, its entries are skipped: %v", err)
	}

	return hosts, conf.Hostnames(), nil
}
