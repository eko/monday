package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestStatusesRegisterAndSet(t *testing.T) {
	// Given
	statuses := NewStatuses(nil)

	// When
	statuses.Register("api", "kubernetes", []string{"8080:8080"})
	statuses.Register("database", "ssh", []string{"5432:5432"})
	statuses.Set("api", StateReady, "forwarding to pod 'api-7f9d4'")

	// Then
	snapshot, version := statuses.snapshot()

	assert.Len(t, snapshot, 2)
	assert.Equal(t, uint64(3), version)

	assert.Equal(t, "api", snapshot[0].Name)
	assert.Equal(t, StateReady, snapshot[0].State)
	assert.Equal(t, "forwarding to pod 'api-7f9d4'", snapshot[0].Message)

	assert.Equal(t, "database", snapshot[1].Name)
	assert.Equal(t, StateWaiting, snapshot[1].State)
}

func TestStatusesAlphabeticalOrder(t *testing.T) {
	// Given
	statuses := NewStatuses(nil)

	// When: registering in a non-alphabetical order
	statuses.Register("wf-api-forward", "kubernetes", []string{"8081:8081"})
	statuses.Register("cast-api-forward", "kubernetes", []string{"8080:8080"})
	statuses.Register("Kafka-broker-forward", "kubernetes", []string{"9092:9092"})

	// Then: the snapshot is sorted alphabetically, case-insensitively
	snapshot, _ := statuses.snapshot()

	assert.Equal(t, "cast-api-forward", snapshot[0].Name)
	assert.Equal(t, "Kafka-broker-forward", snapshot[1].Name)
	assert.Equal(t, "wf-api-forward", snapshot[2].Name)
}

func TestStatusesRegisterTwiceKeepsState(t *testing.T) {
	// Given
	statuses := NewStatuses(nil)
	statuses.Register("api", "kubernetes", []string{"8080:8080"})
	statuses.Set("api", StateReady, "connected")

	// When: a watcher restart re-registers the same forward
	statuses.Register("api", "kubernetes", []string{"8080:8080"})

	// Then
	snapshot, _ := statuses.snapshot()

	assert.Len(t, snapshot, 1)
	assert.Equal(t, StateReady, snapshot[0].State)
}

func TestStatusesSetUnknownForward(t *testing.T) {
	// Given
	statuses := NewStatuses(nil)

	// When
	statuses.Set("unknown", StateReady, "should be ignored")

	// Then
	snapshot, version := statuses.snapshot()

	assert.Len(t, snapshot, 0)
	assert.Equal(t, uint64(0), version)
}

func TestStatusesNotify(t *testing.T) {
	// Given
	statuses := NewStatuses(nil)
	statuses.Register("api", "kubernetes", []string{"8080:8080"})

	notified := 0
	statuses.setNotify(func() {
		notified++
	})

	// When
	statuses.Set("api", StateConnecting, "establishing connection...")
	statuses.Set("api", StateReady, "connected")

	// Then
	assert.Equal(t, 2, notified)
}

func TestStatusesStdoutModePrintsTransitions(t *testing.T) {
	// Given
	stdoutView := NewView("forwards", "Forwards")
	statuses := NewStatuses(stdoutView)
	statuses.Register("api", "kubernetes", []string{"8080:8080"})

	// When: state changes, then only its message changes
	statuses.Set("api", StateReady, "forwarding to pod 'api-1'")
	statuses.Set("api", StateReady, "forwarding to pod 'api-2'")
	statuses.Set("api", StateReconnecting, "pod 'api-2' is terminating")

	// Then: only the 2 state transitions are printed
	lines, _ := stdoutView.snapshot()

	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "Forward 'api' is now ready")
	assert.Contains(t, lines[1], "Forward 'api' is now reconnecting")
}

func newTestCombinedTable() (*combinedTable, *Statuses, *ProxyStatuses) {
	forwards := NewStatuses(nil)
	proxies := NewProxyStatuses()

	return newCombinedTable(forwards, proxies), forwards, proxies
}

func TestCombinedTableLines(t *testing.T) {
	// Given
	table, forwards, proxies := newTestCombinedTable()

	forwards.Register("api-forward", "kubernetes", []string{"8080:8080"})
	forwards.Register("database-forward", "kubernetes", []string{"5432:5432"})
	forwards.Set("api-forward", StateReady, "forwarding to pod 'api-1'")
	forwards.Set("database-forward", StateReconnecting, "lost connection")

	proxies.Register("api-forward", "api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")

	// Local application without forward: displayed as its own row
	proxies.Register("local-app", "app.svc.local", "127.0.1.2", "", "", "")

	// When
	lines := table.tableLines(140, 20, true)
	joined := strings.Join(lines, "\n")

	// Then: summary + header + 3 rows + separator
	assert.Len(t, lines, 6)

	assert.Contains(t, joined, "NAME")
	assert.Contains(t, joined, "STATE")
	assert.Contains(t, joined, "ACTIVITY")
	assert.Contains(t, joined, "LOST")
	assert.Contains(t, joined, "INFO")

	assert.Contains(t, joined, "api-forward")
	assert.Contains(t, joined, "ready")
	assert.Contains(t, joined, "database-forward")
	assert.Contains(t, joined, "reconnecting")
	assert.Contains(t, joined, "local-app")
	assert.Contains(t, joined, "mapped")

	assert.Contains(t, joined, "1 ready")
	assert.Contains(t, joined, "1 reconnecting")
}

func TestCombinedTableSelectionAndExpand(t *testing.T) {
	// Given
	table, forwards, proxies := newTestCombinedTable()

	forwards.Register("api-forward", "kubernetes", []string{"8080:8080"})
	forwards.Register("database-forward", "kubernetes", []string{"5432:5432"})
	proxies.Register("api-forward", "api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")
	proxies.SetState("api.svc.local", "8080", ProxyStateListening, "")
	proxies.ConnectionOpened("api.svc.local", "8080")
	proxies.AddTraffic("api.svc.local", "8080", 2048, 512)

	// When: expanding the first row (api-forward)
	table.toggleSelected()
	joined := strings.Join(table.tableLines(160, 20, true), "\n")

	// Then: proxy details are visible
	assert.Contains(t, joined, "└")
	assert.Contains(t, joined, "api.svc.local")
	assert.Contains(t, joined, "127.0.1.1:8080 → 127.0.0.1:9401")
	assert.Contains(t, joined, "1/1 conn(s)")
	assert.Contains(t, joined, "↓2.0KB ↑512B")

	// When: collapsing again
	table.toggleSelected()
	joined = strings.Join(table.tableLines(160, 20, true), "\n")

	// Then
	assert.NotContains(t, joined, "127.0.1.1:8080")

	// When: selecting a row without proxy details, enter is a no-op
	table.moveSelection(1)
	assert.Equal(t, 1, table.selected)

	table.toggleSelected()
	assert.False(t, table.expanded["database-forward"])

	// Selection is clamped to the rows
	table.moveSelection(10)
	assert.Equal(t, 1, table.selected)
	table.moveSelection(-10)
	assert.Equal(t, 0, table.selected)
}

func TestCombinedTableFilter(t *testing.T) {
	// Given
	table, forwards, proxies := newTestCombinedTable()

	forwards.Register("cast-api-forward", "kubernetes", []string{"8080:8080"})
	forwards.Register("kafka-broker-forward", "kubernetes", []string{"9092:9092"})
	forwards.Register("postgres-forward", "kubernetes", []string{"5432:5432"})
	proxies.Register("kafka-broker-forward", "kafka.svc.local", "127.0.1.2", "9092", "127.0.0.1", "9402")

	// When: filtering by name
	table.setFilter("kafka")
	joined := strings.Join(table.tableLines(140, 20, true), "\n")

	// Then: only the matching row is displayed
	assert.Contains(t, joined, "kafka-broker-forward")
	assert.NotContains(t, joined, "cast-api-forward")
	assert.NotContains(t, joined, "postgres-forward")

	// Selection and expansion follow the filtered rows
	assert.Equal(t, 0, table.selected)
	table.toggleSelected()
	assert.True(t, table.expanded["kafka-broker-forward"])

	// A hostname also matches
	table.setFilter("kafka.svc.local")
	joined = strings.Join(table.tableLines(140, 20, true), "\n")
	assert.Contains(t, joined, "kafka-broker-forward")

	// When: nothing matches
	table.setFilter("unknown")
	joined = strings.Join(table.tableLines(140, 20, true), "\n")

	// Then
	assert.Contains(t, joined, `no forward or hostname matching "unknown"`)

	// The pane visibility is not impacted by the filter
	assert.Equal(t, 3, table.tableCount())

	// When: clearing the filter
	table.setFilter("")
	joined = strings.Join(table.tableLines(140, 20, true), "\n")

	// Then: all rows are back
	assert.Contains(t, joined, "cast-api-forward")
	assert.Contains(t, joined, "postgres-forward")
}

func TestModelFilterAppliesToCombinedTable(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	layout.GetForwardStatuses().Register("cast-api-forward", "kubernetes", []string{"8080:8080"})
	layout.GetForwardStatuses().Register("kafka-broker-forward", "kubernetes", []string{"9092:9092"})

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)
	model.Update(tea.WindowSizeMsg{Width: 130, Height: 40})

	// When: typing "/kafka" then enter on the focused combined pane
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("kafka")})
	model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	rendered := model.View()

	// Then: the table only shows the matching forward
	assert.Contains(t, rendered, "kafka-broker-forward")
	assert.NotContains(t, rendered, "cast-api-forward")

	// When: clearing the filter with esc
	model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	rendered = model.View()

	// Then
	assert.Contains(t, rendered, "cast-api-forward")
}

func TestCombinedTableJumpSelection(t *testing.T) {
	// Given: 8 forwards displayed in a window of 4 rows
	table, forwards, _ := newTestCombinedTable()

	for _, name := range []string{"a-fwd", "b-fwd", "c-fwd", "d-fwd", "e-fwd", "f-fwd", "g-fwd", "h-fwd"} {
		forwards.Register(name, "kubernetes", []string{"8080:8080"})
	}

	// Render once with room for 4 rows (summary + header + 4 rows + separator + indicator)
	table.tableLines(120, 8, true)
	assert.Equal(t, 4, table.visibleCount)

	// When: jumping down from the first row
	table.jumpSelection(1)

	// Then: the selection is on the last visible row
	assert.Equal(t, 3, table.selected)

	// When: jumping down again from the last visible row
	table.jumpSelection(1)

	// Then: a full page down
	assert.Equal(t, 7, table.selected)

	// Jumping down at the end stays clamped
	table.tableLines(120, 8, true)
	table.jumpSelection(1)
	assert.Equal(t, 7, table.selected)

	// When: jumping up, back to the first visible row of the current window
	table.tableLines(120, 8, true)
	table.jumpSelection(-1)
	assert.Equal(t, table.offset, table.selected)

	// Jumping up repeatedly reaches the very first row
	for i := 0; i < 3; i++ {
		table.tableLines(120, 8, true)
		table.jumpSelection(-1)
	}
	assert.Equal(t, 0, table.selected)
}

func TestModelCtrlArrowsJumpSelection(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	for _, name := range []string{"a-fwd", "b-fwd", "c-fwd", "d-fwd"} {
		layout.GetForwardStatuses().Register(name, "kubernetes", []string{"8080:8080"})
	}

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	table := model.focusedPane().table
	if table == nil {
		t.Fatal("expected the combined table to be focused")
	}

	// When: all rows fit, ctrl+down jumps to the last row
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlDown})

	// Then
	assert.Equal(t, 3, table.selected)

	// When: ctrl+up jumps back to the first row
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlUp})

	// Then
	assert.Equal(t, 0, table.selected)

	// The Globe/fn key on macOS sends page down for 🌐+↓: same jump behavior
	model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	assert.Equal(t, 3, table.selected)

	model.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Equal(t, 0, table.selected)

	// 🌐+→ (end) selects the very last row, 🌐+← (home) the first one
	model.Update(tea.KeyMsg{Type: tea.KeyEnd})
	assert.Equal(t, 3, table.selected)

	model.Update(tea.KeyMsg{Type: tea.KeyHome})
	assert.Equal(t, 0, table.selected)
}

func TestCombinedTableWindowFollowsSelection(t *testing.T) {
	// Given: more rows than the table can display
	table, forwards, _ := newTestCombinedTable()

	for _, name := range []string{"a-fwd", "b-fwd", "c-fwd", "d-fwd", "e-fwd", "f-fwd"} {
		forwards.Register(name, "kubernetes", []string{"8080:8080"})
	}

	// When: selecting the last row with a budget of 3 visible rows
	for i := 0; i < 5; i++ {
		table.moveSelection(1)
	}

	joined := strings.Join(table.tableLines(120, 7, true), "\n")

	// Then: the selected row is visible, earlier rows are summarized
	assert.Contains(t, joined, "f-fwd")
	assert.Contains(t, joined, "more above")
	assert.NotContains(t, joined, "a-fwd")
}

func TestModelRendersStatusTable(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	layout.GetForwardStatuses().Register("api", "kubernetes", []string{"8080:8080"})
	layout.GetForwardStatuses().Set("api", StateReady, "forwarding to pod 'api-7f9d4'")

	layout.GetForwardsView().Write("📡  Forwarding 'api' over kubernetes...\n")

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)

	// When
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	rendered := model.View()

	// Then: the state line is displayed (message may be truncated to pane width)
	assert.Contains(t, rendered, "api")
	assert.Contains(t, rendered, "ready")
	assert.Contains(t, rendered, "forwarding to pod")

	// The pane events log is hidden by default, toggled with the "l" key
	assert.NotContains(t, rendered, "Forwarding 'api' over kubernetes")

	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	rendered = model.View()

	assert.Contains(t, rendered, "Forwarding 'api' over kubernetes")
}

func TestStatusesReconnectCounter(t *testing.T) {
	// Given
	statuses := NewStatuses(nil)
	statuses.Register("api", "kubernetes", []string{"8080:8080"})

	// When: two connection losses, with repeated reconnecting updates in between
	statuses.Set("api", StateReconnecting, "lost connection")
	statuses.Set("api", StateReconnecting, "still retrying")
	statuses.Set("api", StateReady, "connected")
	statuses.Set("api", StateReconnecting, "lost connection again")

	// Then
	snapshot, _ := statuses.snapshot()

	assert.Equal(t, 2, snapshot[0].Reconnects)
}

func TestStatusesSample(t *testing.T) {
	// Given
	statuses := NewStatuses(nil)
	statuses.Register("api", "kubernetes", []string{"8080:8080"})
	statuses.Set("api", StateReady, "connected")

	// When
	for i := 0; i < 5; i++ {
		statuses.Sample(3)
	}

	// Then: history is capped to the last 3 samples
	snapshot, _ := statuses.snapshot()

	assert.Equal(t, []ForwardState{StateReady, StateReady, StateReady}, snapshot[0].History)
}

func TestModelSpaceTogglesProxyDetails(t *testing.T) {
	// Given
	layout := NewLayout(true)
	layout.Init()

	layout.GetForwardStatuses().Register("api-forward", "kubernetes", []string{"8080:8080"})
	layout.GetProxyStatuses().Register("api-forward", "api.svc.local", "127.0.1.1", "8080", "127.0.0.1", "9401")

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)
	model.Update(tea.WindowSizeMsg{Width: 140, Height: 40})

	// When: pressing space on the selected row
	model.Update(tea.KeyMsg{Type: tea.KeySpace})
	rendered := model.View()

	// Then: the proxy details are expanded
	assert.Contains(t, rendered, "127.0.1.1:8080 → 127.0.0.1:9401")

	// When: pressing space again
	model.Update(tea.KeyMsg{Type: tea.KeySpace})
	rendered = model.View()

	// Then: collapsed
	assert.NotContains(t, rendered, "127.0.1.1:8080")
}

func TestModelHidesEmptyPanes(t *testing.T) {
	// Given: a forward-only project, no local application logs
	layout := NewLayout(true)
	layout.Init()

	layout.GetForwardStatuses().Register("api", "kubernetes", []string{"8080:8080"})
	layout.GetForwardsView().Write("📡  Forwarding 'api' over kubernetes...\n")
	layout.GetProxyView().Write("✅  Mapped 'api.svc.local'\n")

	model := newModel(
		"my-project",
		"",
		[]*view{layout.logsView, layout.forwardsView},
		layout.forwardStatuses,
		layout.proxyStatuses,
		&layout.dirty,
	)

	// When
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	rendered := model.View()

	// Then: the empty logs pane is hidden and forwards becomes the first pane
	assert.NotContains(t, rendered, "▸ Logs")
	assert.Contains(t, rendered, "1 ▸ Forwards & Proxy")

	// When a local application writes its first log line
	layout.GetLogsView().Write("api starting...\n")
	model.Update(refreshMsg{})
	rendered = model.View()

	// Then: the logs pane appears back
	assert.Contains(t, rendered, "▸ Logs")
}

func TestShortDuration(t *testing.T) {
	testCases := []struct {
		duration time.Duration
		want     string
	}{
		{duration: 12 * time.Second, want: "12s"},
		{duration: 3*time.Minute + 5*time.Second, want: "3m05s"},
		{duration: 2*time.Hour + 7*time.Minute, want: "2h07m"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.want, func(t *testing.T) {
			assert.Equal(t, testCase.want, shortDuration(testCase.duration))
		})
	}
}
