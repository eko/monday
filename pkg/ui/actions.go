package ui

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"

	"github.com/atotto/clipboard"
)

// Actions exposes the operations the terminal UI can trigger on the selected
// forward row
type Actions interface {
	Reconnect(
		name string,
	) error
	TogglePause(
		name string,
	) (bool, error)
	ToggleLogs(
		name string,
	) (bool, error)
}

// handleAction runs the action bound to the given key on the selected row of
// the combined table, reporting feedback into the pane events view
func (m *model) handleAction(key string) {
	pane := m.focusedPane()
	if pane.table == nil {
		return
	}

	row, ok := pane.table.selectedRow()
	if !ok {
		return
	}

	switch key {
	case "r":
		if m.actions == nil {
			return
		}

		if err := m.actions.Reconnect(row.id); err != nil {
			pane.view.Writef("❌  Unable to reconnect '%s': %v\n", row.id, err)
		} else {
			pane.view.Writef("🔁  Reconnection of '%s' requested\n", row.id)
		}

	case "p":
		if m.actions == nil {
			return
		}

		paused, err := m.actions.TogglePause(row.id)
		switch {
		case err != nil:
			pane.view.Writef("❌  Unable to pause/resume '%s': %v\n", row.id, err)
		case paused:
			pane.view.Writef("⏸  Forward '%s' has been paused\n", row.id)
		default:
			pane.view.Writef("▶️  Forward '%s' has been resumed\n", row.id)
		}

	case "L":
		if m.actions == nil {
			return
		}

		if _, err := m.actions.ToggleLogs(row.id); err != nil {
			pane.view.Writef("❌  Unable to stream pod logs of '%s': %v\n", row.id, err)
		}

	case "c":
		address, ok := rowAddress(row)
		if !ok {
			pane.view.Writef("❌  No hostname to copy for '%s'\n", row.id)
			return
		}

		if err := clipboard.WriteAll(address); err != nil {
			pane.view.Writef("❌  Unable to copy '%s' to the clipboard: %v\n", address, err)
		} else {
			pane.view.Writef("📋  Copied '%s' to the clipboard\n", address)
		}

	case "o":
		address, ok := rowAddress(row)
		if !ok {
			pane.view.Writef("❌  No hostname to open for '%s'\n", row.id)
			return
		}

		url := "http://" + address

		if err := openBrowser(url); err != nil {
			pane.view.Writef("❌  Unable to open '%s': %v\n", url, err)
		} else {
			pane.view.Writef("🌍  Opening '%s' in your browser\n", url)
		}
	}

	m.refreshContents()
}

// rowAddress returns the local address of the selected row: its first proxied
// hostname along with its port when one is proxified
func rowAddress(row combinedRow) (string, bool) {
	if len(row.proxies) == 0 {
		return "", false
	}

	proxyStatus := row.proxies[0]

	if proxyStatus.LocalPort == "" {
		return proxyStatus.Hostname, true
	}

	return net.JoinHostPort(proxyStatus.Hostname, proxyStatus.LocalPort), true
}

// openBrowser opens the given URL in the default browser
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("unsupported operating system (%s)", runtime.GOOS)
	}
}
