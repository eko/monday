package ui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/reflow/wrap"
)

const (
	bottomPanesMinHeight = 8
	scrollStep           = 3

	// tickInterval is the base refresh rate of the terminal UI, content changes
	// refresh the UI immediately regardless
	tickInterval = 500 * time.Millisecond

	// sampleEveryTicks is the number of ticks between two state history samples
	sampleEveryTicks = 4
)

var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#5A3FD6", Dark: "#9D86F9"}
	colorBorder = lipgloss.AdaptiveColor{Light: "#D0CCE3", Dark: "#3B3B54"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#8A87A0", Dark: "#6E6A86"}
	colorTitle  = lipgloss.AdaptiveColor{Light: "#2D2A45", Dark: "#E0DEF4"}
	colorOk     = lipgloss.AdaptiveColor{Light: "#2E7D32", Dark: "#9CCC65"}

	headerBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorAccent).
				Padding(0, 1)

	headerProjectStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorTitle).
				Padding(0, 1)

	headerInfoStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	paneFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent)

	paneTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTitle)

	paneTitleFocusedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	paneInfoStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	footerKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	footerDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	autoscrollOnStyle = lipgloss.NewStyle().
				Foreground(colorOk)

	filterPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	colorWarn  = lipgloss.AdaptiveColor{Light: "#B26A00", Dark: "#E5C07B"}
	colorError = lipgloss.AdaptiveColor{Light: "#C62828", Dark: "#E06C75"}

	stateStyles = map[ForwardState]lipgloss.Style{
		StateWaiting:      lipgloss.NewStyle().Foreground(colorMuted),
		StateConnecting:   lipgloss.NewStyle().Foreground(colorWarn),
		StateReady:        lipgloss.NewStyle().Foreground(colorOk),
		StateReconnecting: lipgloss.NewStyle().Foreground(colorWarn),
		StateError:        lipgloss.NewStyle().Foreground(colorError),
		StateStopped:      lipgloss.NewStyle().Foreground(colorMuted),
	}

	statusNameStyle = lipgloss.NewStyle().Bold(true).Foreground(colorTitle)
	statusMetaStyle = lipgloss.NewStyle().Foreground(colorMuted)

	statusSeparatorStyle = lipgloss.NewStyle().Foreground(colorBorder)

	// stateSparks are the graph characters used to draw the state history of
	// each forward, one sample per character
	stateSparks = map[ForwardState]string{
		StateWaiting:      "▁",
		StateConnecting:   "▄",
		StateReady:        "▇",
		StateReconnecting: "▄",
		StateError:        "▂",
		StateStopped:      "▁",
	}
)

// sparklineWidth is the number of state samples drawn per forward
const sparklineWidth = 20

type refreshMsg struct{}

type tickMsg time.Time

// pane wraps a buffered view into a scrollable terminal UI panel
type pane struct {
	view        *view
	viewport    viewport.Model
	autoscroll  bool
	lineCount   int
	lastVersion uint64
	lastWidth   int
	lastFilter  string
	initialized bool

	// table, when set, is rendered as an interactive table pinned above the
	// pane content
	table        *combinedTable
	statusRender string
	statusLines  int
	baseHeight   int

	// showLog displays the events log below the table, toggled with the "l" key
	showLog bool
}

// model is the Bubble Tea model rendering the Monday terminal UI
type model struct {
	project   string
	version   string
	startedAt time.Time

	panes []*pane
	focus int

	// visible holds the indexes of the panes having content: panes without any
	// activity (e.g. logs on a forward-only project) are not displayed
	visible []int

	statuses  *Statuses
	tickCount int

	fullscreen  bool
	filtering   bool
	filter      string
	filterInput textinput.Model
	showAbout   bool

	width  int
	height int
	ready  bool

	dirty *atomic.Bool
}

func newModel(
	project, version string,
	views []*view,
	statuses *Statuses,
	proxyStatuses *ProxyStatuses,
	dirty *atomic.Bool,
) *model {
	panes := make([]*pane, 0, len(views))
	for _, v := range views {
		p := &pane{
			view:       v,
			autoscroll: true,
		}

		if v.GetName() == "activity" && statuses != nil && proxyStatuses != nil {
			p.table = newCombinedTable(statuses, proxyStatuses)
		}

		panes = append(panes, p)
	}

	filterInput := textinput.New()
	filterInput.Placeholder = "filter lines..."
	filterInput.Prompt = "/ "
	filterInput.PromptStyle = filterPromptStyle
	filterInput.CharLimit = 64

	return &model{
		project:     project,
		version:     version,
		startedAt:   time.Now(),
		panes:       panes,
		statuses:    statuses,
		filterInput: filterInput,
		dirty:       dirty,
	}
}

func (m *model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.resizePanes()
		m.refreshContents()

	case tickMsg:
		m.tickCount++
		if m.statuses != nil && m.tickCount%sampleEveryTicks == 0 {
			m.statuses.Sample(sparklineWidth)
		}

		m.refreshContents()
		return m, tick()

	case refreshMsg:
		if m.dirty != nil {
			m.dirty.Store(false)
		}
		m.refreshContents()

	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			break
		}

		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.scrollFocused(-scrollStep)
		case tea.MouseButtonWheelDown:
			m.scrollFocused(scrollStep)
		}

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFiltering(msg)
		}

		if m.showAbout {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}

			m.showAbout = false

			return m, nil
		}

		return m.updateKeys(msg)
	}

	return m, nil
}

func (m *model) updateFiltering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter = ""
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.refreshContents()
		return m, nil

	case "enter":
		m.filtering = false
		m.filterInput.Blur()
		return m, nil

	case "ctrl+c":
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filter = m.filterInput.Value()
	m.refreshContents()

	return m, cmd
}

func (m *model) updateKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab", "right":
		m.focusNextPane(1)

	case "shift+tab", "left":
		m.focusNextPane(-1)

	case "1", "2", "3":
		visible := m.visiblePanes()

		index := int(msg.String()[0] - '1')
		if index < len(visible) {
			m.focusPane(visible[index])
		}

	case "up":
		// In the combined table pane, arrows move the row selection while j/k
		// keep scrolling the events log below
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.moveSelection(-1)
			m.refreshContents()
		} else {
			m.scrollFocused(-scrollStep)
		}

	case "down":
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.moveSelection(1)
			m.refreshContents()
		} else {
			m.scrollFocused(scrollStep)
		}

	case "ctrl+up":
		// Jump to the first visible row, then a full page up
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.jumpSelection(-1)
			m.refreshContents()
		} else {
			m.scrollFocused(-m.focusedPane().viewport.Height)
		}

	case "ctrl+down":
		// Jump to the last visible row, then a full page down
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.jumpSelection(1)
			m.refreshContents()
		} else {
			m.scrollFocused(m.focusedPane().viewport.Height)
		}

	case "enter", " ":
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.toggleSelected()
			m.refreshContents()
		}

	case "l":
		if pane := m.focusedPane(); pane.table != nil {
			pane.showLog = !pane.showLog
			if pane.showLog {
				pane.autoscroll = true
				pane.viewport.GotoBottom()
			}
			m.refreshContents()
		}

	case "k":
		m.scrollFocused(-scrollStep)

	case "j":
		m.scrollFocused(scrollStep)

	// The Globe/fn key on macOS sends page up/down for 🌐+↑/↓
	case "pgup":
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.jumpSelection(-1)
			m.refreshContents()
		} else {
			m.scrollFocused(-m.focusedPane().viewport.Height)
		}

	case "pgdown":
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.jumpSelection(1)
			m.refreshContents()
		} else {
			m.scrollFocused(m.focusedPane().viewport.Height)
		}

	// The Globe/fn key on macOS sends home/end for 🌐+←/→
	case "g", "home":
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.moveSelection(-pane.table.tableCount())
			m.refreshContents()
		} else {
			pane.autoscroll = false
			pane.viewport.GotoTop()
		}

	case "G", "end":
		if pane := m.focusedPane(); pane.table != nil {
			pane.table.moveSelection(pane.table.tableCount())
			m.refreshContents()
		} else {
			pane.autoscroll = true
			pane.viewport.GotoBottom()
		}

	case "a":
		pane := m.focusedPane()
		pane.autoscroll = !pane.autoscroll
		if pane.autoscroll {
			pane.viewport.GotoBottom()
		}

	case "f":
		m.fullscreen = !m.fullscreen
		m.resizePanes()
		m.refreshContents()

	case "?":
		m.showAbout = true

	case "/":
		m.filtering = true
		m.filterInput.Focus()
		return m, textinput.Blink

	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.filterInput.SetValue("")
			m.refreshContents()
		} else if m.fullscreen {
			m.fullscreen = false
			m.resizePanes()
			m.refreshContents()
		}
	}

	return m, nil
}

func (m *model) focusedPane() *pane {
	return m.panes[m.focus]
}

func (m *model) focusPane(index int) {
	m.focus = index
	m.resizePanes()
	m.refreshContents()
}

// focusNextPane moves the focus among the visible panes in the given direction
func (m *model) focusNextPane(direction int) {
	visible := m.visiblePanes()

	position := 0
	for i, index := range visible {
		if index == m.focus {
			position = i
			break
		}
	}

	position = (position + direction + len(visible)) % len(visible)

	m.focusPane(visible[position])
}

// visiblePanes returns the indexes of the panes having content, or all of them
// when none has content yet
func (m *model) visiblePanes() []int {
	if len(m.visible) > 0 {
		return m.visible
	}

	all := make([]int, len(m.panes))
	for i := range all {
		all[i] = i
	}

	return all
}

// refreshVisibility recomputes which panes have content, adapting the layout
// and the focus when a pane appears or disappears
func (m *model) refreshVisibility() {
	visible := make([]int, 0, len(m.panes))

	for i, pane := range m.panes {
		_, version := pane.view.snapshot()

		hasTable := pane.table != nil && pane.table.tableCount() > 0

		if version > 0 || hasTable {
			visible = append(visible, i)
		}
	}

	if slicesEqual(visible, m.visible) {
		return
	}

	m.visible = visible

	focusVisible := false
	for _, index := range m.visiblePanes() {
		if index == m.focus {
			focusVisible = true
			break
		}
	}

	if !focusVisible {
		m.focus = m.visiblePanes()[0]
	}

	m.resizePanes()
}

func slicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func (m *model) scrollFocused(lines int) {
	pane := m.focusedPane()

	if lines < 0 {
		pane.autoscroll = false
	}

	pane.viewport.SetYOffset(pane.viewport.YOffset + lines)

	if pane.viewport.AtBottom() {
		pane.autoscroll = true
	}
}

// paneSizes returns the outer box width/height of each pane depending on the
// current terminal size, visible panes, focus and fullscreen state
func (m *model) paneSizes() [][2]int {
	sizes := make([][2]int, len(m.panes))

	contentHeight := m.height - m.headerHeight() - m.footerHeight()
	if contentHeight < 4 {
		contentHeight = 4
	}

	if m.fullscreen {
		sizes[m.focus] = [2]int{m.width, contentHeight}

		return sizes
	}

	visible := m.visiblePanes()

	bottomHeight := contentHeight / 3
	if bottomHeight < bottomPanesMinHeight {
		bottomHeight = bottomPanesMinHeight
	}
	if bottomHeight > contentHeight-4 {
		bottomHeight = contentHeight - 4
	}

	switch len(visible) {
	case 1:
		sizes[visible[0]] = [2]int{m.width, contentHeight}

	case 2:
		// Both panes hold live tables: share the space equally
		topHeight := contentHeight / 2
		sizes[visible[0]] = [2]int{m.width, topHeight}
		sizes[visible[1]] = [2]int{m.width, contentHeight - topHeight}

	default:
		leftWidth := m.width / 2
		rightWidth := m.width - leftWidth

		sizes[visible[0]] = [2]int{m.width, contentHeight - bottomHeight}
		sizes[visible[1]] = [2]int{leftWidth, bottomHeight}
		sizes[visible[2]] = [2]int{rightWidth, bottomHeight}
	}

	return sizes
}

func (m *model) resizePanes() {
	if !m.ready {
		return
	}

	for i, size := range m.paneSizes() {
		pane := m.panes[i]

		width, height := size[0], size[1]
		if width == 0 || height == 0 {
			continue
		}

		// Borders take 2 columns/rows, the pane title one more line
		innerWidth := width - 2
		innerHeight := height - 3

		if innerWidth < 1 {
			innerWidth = 1
		}
		if innerHeight < 1 {
			innerHeight = 1
		}

		pane.baseHeight = innerHeight

		viewportHeight := innerHeight - pane.statusLines
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		if !pane.initialized {
			pane.viewport = viewport.New(innerWidth, viewportHeight)
			pane.initialized = true
		} else {
			pane.viewport.Width = innerWidth
			pane.viewport.Height = viewportHeight
		}
	}
}

func (m *model) refreshContents() {
	if !m.ready {
		return
	}

	m.refreshVisibility()

	for i, pane := range m.panes {
		m.refreshStatusTable(pane)

		filter := ""
		if i == m.focus {
			filter = m.filter
		}

		lines, version := pane.view.snapshot()

		if version == pane.lastVersion &&
			pane.viewport.Width == pane.lastWidth &&
			filter == pane.lastFilter {
			continue
		}

		pane.lastVersion = version
		pane.lastWidth = pane.viewport.Width
		pane.lastFilter = filter

		if filter != "" {
			lines = filterLines(lines, filter)
		}

		pane.lineCount = len(lines)
		pane.viewport.SetContent(wrap.String(strings.Join(lines, "\n"), pane.viewport.Width))

		if pane.autoscroll {
			pane.viewport.GotoBottom()
		}
	}
}

// refreshStatusTable rebuilds the live table pinned above the pane content,
// shrinking the pane viewport accordingly
func (m *model) refreshStatusTable(pane *pane) {
	if pane.table == nil || !pane.initialized {
		return
	}

	focused := m.panes[m.focus] == pane

	filter := ""
	if focused {
		filter = m.filter
	}
	pane.table.setFilter(filter)

	// The table gets the whole pane when the events log is hidden
	maxLines := pane.baseHeight
	if pane.showLog {
		maxLines = pane.baseHeight - 2
	}

	lines := pane.table.tableLines(pane.viewport.Width, maxLines, focused)
	pane.statusRender = strings.Join(lines, "\n")

	if len(lines) != pane.statusLines {
		pane.statusLines = len(lines)

		viewportHeight := pane.baseHeight - pane.statusLines
		if viewportHeight < 1 {
			viewportHeight = 1
		}

		pane.viewport.Height = viewportHeight
	}
}

var proxyStateStyles = map[ProxyState]lipgloss.Style{
	ProxyStateWaiting:   lipgloss.NewStyle().Foreground(colorMuted),
	ProxyStateMapped:    lipgloss.NewStyle().Foreground(colorAccent),
	ProxyStateListening: lipgloss.NewStyle().Foreground(colorOk),
	ProxyStateError:     lipgloss.NewStyle().Foreground(colorError),
}

var proxyStateIcons = map[ProxyState]string{
	ProxyStateWaiting:   "○",
	ProxyStateMapped:    "◆",
	ProxyStateListening: "●",
	ProxyStateError:     "✕",
}

// renderSparkline renders the recent state history of a forward as a mini
// graph: high green blocks when ready, medium yellow when (re)connecting, low
// red on errors
func renderSparkline(history []ForwardState) string {
	if len(history) > sparklineWidth {
		history = history[len(history)-sparklineWidth:]
	}

	var sparkline strings.Builder

	for i := 0; i < sparklineWidth-len(history); i++ {
		sparkline.WriteString(statusMetaStyle.Render("·"))
	}

	for _, state := range history {
		sparkline.WriteString(stateStyles[state].Render(stateSparks[state]))
	}

	return sparkline.String()
}

// truncateLine hard-truncates a rendered line to the given width, accounting
// for ANSI escape sequences
func truncateLine(line string, width int) string {
	return truncate.StringWithTail(line, uint(width), "…")
}

func filterLines(lines []string, filter string) []string {
	filtered := make([]string, 0, len(lines))
	lowered := strings.ToLower(filter)

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), lowered) {
			filtered = append(filtered, line)
		}
	}

	return filtered
}

func (m *model) headerHeight() int {
	return 1
}

func (m *model) footerHeight() int {
	return 1
}

func (m *model) View() string {
	if !m.ready {
		return "Loading Monday..."
	}

	sections := []string{m.headerView()}

	if m.showAbout {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.headerView(),
			m.aboutView(),
			m.footerView(),
		)
	}

	if m.fullscreen {
		sections = append(sections, m.paneView(m.focus))
	} else {
		visible := m.visiblePanes()

		switch len(visible) {
		case 1:
			sections = append(sections, m.paneView(visible[0]))

		case 2:
			sections = append(sections, m.paneView(visible[0]), m.paneView(visible[1]))

		default:
			sections = append(
				sections,
				m.paneView(visible[0]),
				lipgloss.JoinHorizontal(lipgloss.Top, m.paneView(visible[1]), m.paneView(visible[2])),
			)
		}
	}

	sections = append(sections, m.footerView())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m *model) headerView() string {
	badge := headerBadgeStyle.Render("⚡ Monday " + m.version)
	project := headerProjectStyle.Render(m.project)

	uptime := time.Since(m.startedAt).Round(time.Second)
	info := headerInfoStyle.Render(fmt.Sprintf("uptime %s ", uptime))

	left := lipgloss.JoinHorizontal(lipgloss.Center, badge, project)

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(info)
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + info
}

func (m *model) paneView(index int) string {
	pane := m.panes[index]
	sizes := m.paneSizes()
	width, height := sizes[index][0], sizes[index][1]

	if width == 0 || height == 0 || !pane.initialized {
		return ""
	}

	boxStyle, titleStyle := paneStyle, paneTitleStyle
	if index == m.focus {
		boxStyle, titleStyle = paneFocusedStyle, paneTitleFocusedStyle
	}

	number := 1
	for position, visibleIndex := range m.visiblePanes() {
		if visibleIndex == index {
			number = position + 1
			break
		}
	}

	title := titleStyle.Render(fmt.Sprintf("%d ▸ %s", number, pane.view.GetTitle()))

	logHidden := pane.table != nil && !pane.showLog

	var details string
	if logHidden {
		details = fmt.Sprintf("%d event(s) · press l for logs", pane.lineCount)
	} else {
		details = fmt.Sprintf("%d line(s) · %3.f%%", pane.lineCount, pane.viewport.ScrollPercent()*100)
		if pane.autoscroll {
			details += " · " + autoscrollOnStyle.Render("follow")
		}
	}
	if index == m.focus && m.filter != "" {
		details = fmt.Sprintf("filter: %q · %s", m.filter, details)
	}

	info := paneInfoStyle.Render(details)

	innerWidth := width - 2
	gap := innerWidth - lipgloss.Width(title) - lipgloss.Width(info) - 2
	if gap < 1 {
		gap = 1
		info = ""
	}

	titleBar := " " + title + strings.Repeat(" ", gap) + info + " "

	sections := []string{titleBar}
	if pane.statusLines > 0 {
		sections = append(sections, pane.statusRender)
	}

	if logHidden {
		// Keep the pane height stable by padding the space left by the table
		if padding := pane.baseHeight - pane.statusLines; padding > 0 {
			sections = append(sections, strings.Repeat("\n", padding-1))
		}
	} else {
		sections = append(sections, pane.viewport.View())
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	return boxStyle.Width(innerWidth).Render(content)
}

func (m *model) footerView() string {
	if m.filtering {
		return " " + m.filterInput.View()
	}

	if m.showAbout {
		return footerDescStyle.Render(" press any key to close")
	}

	var keys [][2]string

	if pane := m.focusedPane(); pane.table != nil {
		keys = [][2]string{
			{"tab", "switch"},
			{"↑/↓", "select"},
			{"⌃/🌐↑↓", "jump"},
			{"⏎/␣", "details"},
			{"l", "logs"},
			{"f", "fullscreen"},
			{"/", "filter"},
			{"?", "about"},
			{"q", "quit"},
		}

		if pane.showLog {
			keys = append(keys[:5:5], append([][2]string{{"j/k", "scroll"}, {"a", "follow"}}, keys[5:]...)...)
		}
	} else {
		keys = [][2]string{
			{"tab", "switch"},
			{"↑/↓", "scroll"},
			{"a", "follow"},
			{"f", "fullscreen"},
			{"/", "filter"},
			{"?", "about"},
			{"q", "quit"},
		}
	}

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, footerKeyStyle.Render(key[0])+footerDescStyle.Render(" "+key[1]))
	}

	return " " + strings.Join(parts, footerDescStyle.Render(" · "))
}
