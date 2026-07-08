package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#2D2A45", Dark: "#FFFFFF"}).
				Background(lipgloss.AdaptiveColor{Light: "#E9E4FB", Dark: "#3B3560"})

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorMuted)

	detailBranchStyle = lipgloss.NewStyle().
				Foreground(colorBorder)
)

// combinedTable renders the forwards and their proxy routing details as a
// single interactive table: rows can be selected and expanded to reveal the
// proxy details of each forward
type combinedTable struct {
	forwards *Statuses
	proxies  *ProxyStatuses

	selected int
	offset   int
	expanded map[string]bool
}

func newCombinedTable(
	forwards *Statuses,
	proxies *ProxyStatuses,
) *combinedTable {
	return &combinedTable{
		forwards: forwards,
		proxies:  proxies,
		expanded: make(map[string]bool),
	}
}

// combinedRow is a selectable table row: a forward along with its proxy
// entries, or a proxy-only entry (e.g. a local application hostname)
type combinedRow struct {
	id      string
	forward *ForwardStatus
	proxies []ProxyStatus
}

// rows builds the selectable rows: every forward first, then proxy-only entries
func (t *combinedTable) rows() []combinedRow {
	forwards, _ := t.forwards.snapshot()
	proxies, _ := t.proxies.snapshot()

	proxiesByName := make(map[string][]ProxyStatus)
	for _, proxyStatus := range proxies {
		proxiesByName[proxyStatus.Name] = append(proxiesByName[proxyStatus.Name], proxyStatus)
	}

	rows := make([]combinedRow, 0, len(forwards))
	seen := make(map[string]bool, len(forwards))

	for i := range forwards {
		forward := forwards[i]
		seen[forward.Name] = true

		rows = append(rows, combinedRow{
			id:      forward.Name,
			forward: &forward,
			proxies: proxiesByName[forward.Name],
		})
	}

	// Proxy-only entries: hostnames not attached to any forward (local applications)
	proxyOnlyNames := make([]string, 0)
	for _, proxyStatus := range proxies {
		if seen[proxyStatus.Name] {
			continue
		}

		seen[proxyStatus.Name] = true
		proxyOnlyNames = append(proxyOnlyNames, proxyStatus.Name)
	}

	sort.Slice(proxyOnlyNames, func(i, j int) bool {
		return strings.ToLower(proxyOnlyNames[i]) < strings.ToLower(proxyOnlyNames[j])
	})

	for _, name := range proxyOnlyNames {
		rows = append(rows, combinedRow{
			id:      name,
			proxies: proxiesByName[name],
		})
	}

	return rows
}

func (t *combinedTable) tableCount() int {
	return len(t.rows())
}

// moveSelection moves the selected row by the given delta, clamped to the rows
func (t *combinedTable) moveSelection(delta int) {
	count := len(t.rows())
	if count == 0 {
		return
	}

	t.selected += delta

	if t.selected < 0 {
		t.selected = 0
	}
	if t.selected >= count {
		t.selected = count - 1
	}
}

// toggleSelected expands or collapses the proxy details of the selected row
func (t *combinedTable) toggleSelected() {
	rows := t.rows()
	if t.selected >= len(rows) {
		return
	}

	row := rows[t.selected]
	if len(row.proxies) == 0 {
		return
	}

	t.expanded[row.id] = !t.expanded[row.id]
}

// tableLines renders the table: a summary, column headers, the (windowed)
// selectable rows with their expanded details, and a closing separator
func (t *combinedTable) tableLines(width, maxLines int, focused bool) []string {
	rows := t.rows()

	if len(rows) == 0 || width < 10 || maxLines < 4 {
		return nil
	}

	if t.selected >= len(rows) {
		t.selected = len(rows) - 1
	}

	nameWidth := 0
	for _, row := range rows {
		if len(row.id) > nameWidth {
			nameWidth = len(row.id)
		}
	}

	// Render each row block: the main line plus its details when expanded
	blocks := make([][]string, 0, len(rows))
	for i, row := range rows {
		selected := focused && i == t.selected

		block := []string{t.renderMainLine(row, nameWidth, width, selected)}

		if t.expanded[row.id] {
			for _, proxyStatus := range row.proxies {
				block = append(block, renderProxyDetail(proxyStatus, width))
			}
		}

		blocks = append(blocks, block)
	}

	budget := maxLines - 3
	windowed, hiddenAbove, hiddenBelow := t.window(blocks, budget)

	lines := make([]string, 0, maxLines)
	lines = append(lines, renderCombinedSummary(rows, width))
	lines = append(lines, renderCombinedHeader(nameWidth, width))

	if hiddenAbove > 0 {
		lines = append(lines, statusMetaStyle.Render(fmt.Sprintf("   ↑ %d more above", hiddenAbove)))
	}

	lines = append(lines, windowed...)

	if hiddenBelow > 0 {
		lines = append(lines, statusMetaStyle.Render(fmt.Sprintf("   ↓ %d more below", hiddenBelow)))
	}

	lines = append(lines, statusSeparatorStyle.Render(strings.Repeat("─", width)))

	return lines
}

// window keeps the selected row visible inside the given line budget, hiding
// rows above and below when the table does not fit
func (t *combinedTable) window(blocks [][]string, budget int) ([]string, int, int) {
	total := 0
	for _, block := range blocks {
		total += len(block)
	}

	if total <= budget {
		t.offset = 0

		lines := make([]string, 0, total)
		for _, block := range blocks {
			lines = append(lines, block...)
		}

		return lines, 0, 0
	}

	if t.offset > t.selected {
		t.offset = t.selected
	}
	if t.offset < 0 {
		t.offset = 0
	}

	// Grow the offset until the selected block fits in the budget, keeping room
	// for the hidden rows indicators
	for {
		used := 0
		if t.offset > 0 {
			used++
		}
		used++ // Room for the "more below" indicator

		fits := true
		for i := t.offset; i <= t.selected && i < len(blocks); i++ {
			used += len(blocks[i])
			if used > budget {
				fits = false
				break
			}
		}

		if fits || t.offset >= t.selected {
			break
		}

		t.offset++
	}

	lines := make([]string, 0, budget)
	used := 0
	if t.offset > 0 {
		used++
	}

	shown := t.offset
	for i := t.offset; i < len(blocks); i++ {
		reserve := 0
		if i < len(blocks)-1 {
			reserve = 1
		}

		if used+len(blocks[i])+reserve > budget && i > t.offset {
			break
		}

		lines = append(lines, blocks[i]...)
		used += len(blocks[i])
		shown++
	}

	return lines, t.offset, len(blocks) - shown
}

// renderMainLine renders a selectable row: a forward line (kept identical to
// the historical format) or a proxy-only application line
func (t *combinedTable) renderMainLine(row combinedRow, nameWidth, width int, selected bool) string {
	chevron := " "
	if len(row.proxies) > 0 {
		chevron = "▸"
		if t.expanded[row.id] {
			chevron = "▾"
		}
	}

	var icon, state, sparkline, plainSparkline, reconnects, details string

	if row.forward != nil {
		forward := row.forward
		stateStyle := stateStyles[forward.State]

		icon = stateStyle.Render(stateIcons[forward.State])
		state = stateStyle.Render(fmt.Sprintf("%-12s", forward.State))
		sparkline = renderSparkline(forward.History)
		plainSparkline = renderPlainSparkline(forward.History)
		reconnects = statusMetaStyle.Render(fmt.Sprintf("↻%-3d", forward.Reconnects))

		details = forward.Message
		if details != "" {
			details += " · "
		}
		details += shortDuration(time.Since(forward.UpdatedAt))

		if selected {
			return t.renderSelectedLine(chevron, stateIcons[forward.State], row.id, string(forward.State), plainSparkline, fmt.Sprintf("↻%-3d", forward.Reconnects), details, nameWidth, width)
		}
	} else {
		proxyStatus := row.proxies[0]
		stateStyle := proxyStateStyles[proxyStatus.State]

		icon = stateStyle.Render(proxyStateIcons[proxyStatus.State])
		state = stateStyle.Render(fmt.Sprintf("%-12s", proxyStatus.State))
		sparkline = renderSparkline(nil)
		plainSparkline = renderPlainSparkline(nil)
		reconnects = statusMetaStyle.Render("    ")
		details = proxyStatus.Hostname

		if selected {
			return t.renderSelectedLine(chevron, proxyStateIcons[proxyStatus.State], row.id, string(proxyStatus.State), plainSparkline, "    ", details, nameWidth, width)
		}
	}

	line := fmt.Sprintf(
		" %s %s %s  %s  %s %s %s",
		detailBranchStyle.Render(chevron),
		icon,
		statusNameStyle.Render(fmt.Sprintf("%-*s", nameWidth, row.id)),
		state,
		sparkline,
		reconnects,
		statusMetaStyle.Render(details),
	)

	return truncateLine(line, width)
}

// renderSelectedLine renders the highlighted selected row: a single style over
// plain text so the background stays solid across the whole line
func (t *combinedTable) renderSelectedLine(
	chevron, icon, name, state, sparkline, reconnects, details string,
	nameWidth, width int,
) string {
	line := fmt.Sprintf(
		" %s %s %s  %-12s  %s %s %s",
		chevron,
		icon,
		fmt.Sprintf("%-*s", nameWidth, name),
		state,
		sparkline,
		reconnects,
		details,
	)

	if padding := width - lipgloss.Width(line); padding > 0 {
		line += strings.Repeat(" ", padding)
	}

	return selectedRowStyle.Render(truncateLine(line, width))
}

// renderProxyDetail renders an expanded proxy entry of a row: its routing,
// connections and live traffic
func renderProxyDetail(status ProxyStatus, width int) string {
	stateStyle := proxyStateStyles[status.State]

	line := fmt.Sprintf(
		"     %s %s %s %s  %s  %s",
		detailBranchStyle.Render("└"),
		stateStyle.Render(proxyStateIcons[status.State]),
		stateStyle.Render(string(status.State)),
		statusNameStyle.Render(status.Hostname),
		statusMetaStyle.Render(renderProxyRoute(status)),
		statusMetaStyle.Render(fmt.Sprintf(
			"%d/%d conn(s) · ↓%s ↑%s",
			status.ActiveConnections,
			status.TotalConnections,
			humanBytes(status.BytesReceived),
			humanBytes(status.BytesSent),
		)),
	)

	if status.Errors > 0 {
		line += " " + stateStyles[StateError].Render(fmt.Sprintf("⚠%d", status.Errors))
	}

	if status.State == ProxyStateError && status.Message != "" {
		line += " " + statusMetaStyle.Render(status.Message)
	}

	return truncateLine(line, width)
}

// renderProxyRoute renders the local address and its forwarded target
func renderProxyRoute(status ProxyStatus) string {
	if status.LocalPort == "" {
		return fmt.Sprintf("%s (hostname only)", status.LocalIP)
	}

	return fmt.Sprintf(
		"%s:%s → %s:%s",
		status.LocalIP,
		status.LocalPort,
		status.TargetHost,
		status.TargetPort,
	)
}

// renderCombinedHeader renders the table column names, aligned with the rows
func renderCombinedHeader(nameWidth, width int) string {
	header := fmt.Sprintf(
		"     %-*s  %-12s  %-*s %-4s %s",
		nameWidth,
		"NAME",
		"STATE",
		sparklineWidth,
		"ACTIVITY",
		"LOST",
		"INFO",
	)

	return tableHeaderStyle.Render(truncateLine(header, width))
}

// renderCombinedSummary renders the aggregated forward states, lost
// connections, active proxy connections and total transferred bytes
func renderCombinedSummary(rows []combinedRow, width int) string {
	counts := map[ForwardState]int{}
	reconnects, active := 0, 0
	received, sent := int64(0), int64(0)

	for _, row := range rows {
		if row.forward != nil {
			counts[row.forward.State]++
			reconnects += row.forward.Reconnects
		}

		for _, proxyStatus := range row.proxies {
			active += proxyStatus.ActiveConnections
			received += proxyStatus.BytesReceived
			sent += proxyStatus.BytesSent
		}
	}

	parts := make([]string, 0, 6)

	for _, state := range []ForwardState{StateReady, StateConnecting, StateReconnecting, StateError, StateWaiting, StateStopped} {
		if counts[state] == 0 {
			continue
		}

		parts = append(parts, stateStyles[state].Render(fmt.Sprintf("%s %d %s", stateIcons[state], counts[state], state)))
	}

	summary := " " + strings.Join(parts, statusMetaStyle.Render(" · "))
	summary += statusMetaStyle.Render(fmt.Sprintf(
		" · ↻ %d lost · %d active conn(s) · ↓%s ↑%s",
		reconnects,
		active,
		humanBytes(received),
		humanBytes(sent),
	))

	return truncateLine(summary, width)
}

// renderPlainSparkline renders the state history without colors, used inside
// the selected row so its background highlight stays solid
func renderPlainSparkline(history []ForwardState) string {
	if len(history) > sparklineWidth {
		history = history[len(history)-sparklineWidth:]
	}

	var sparkline strings.Builder

	for i := 0; i < sparklineWidth-len(history); i++ {
		sparkline.WriteString("·")
	}

	for _, state := range history {
		sparkline.WriteString(stateSparks[state])
	}

	return sparkline.String()
}
