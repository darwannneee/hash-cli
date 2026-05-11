package tui

import (
	"sort"

	"hashminer/internal/miner"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	tabs     []string
	active   int
	snapshot miner.Snapshot
	logs     []miner.Event
	settings map[string]string
	actions  chan Action
	updates  <-chan miner.Snapshot
	events   <-chan miner.Event
	help     bool
}

type snapshotMsg miner.Snapshot
type eventMsg miner.Event

func New(updates <-chan miner.Snapshot, events <-chan miner.Event, actions chan Action, settings map[string]string) Model {
	return Model{
		tabs:     []string{"Dashboard", "Mining", "Chain", "Transactions", "Logs", "Settings"},
		actions:  actions,
		updates:  updates,
		events:   events,
		settings: settings,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(waitSnapshot(m.updates), waitEvent(m.events))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		switch v.String() {
		case "ctrl+c", "q":
			m.send(ActionQuit)
			return m, tea.Quit
		case "right", "l":
			m.active = (m.active + 1) % len(m.tabs)
		case "left", "h":
			m.active--
			if m.active < 0 {
				m.active = len(m.tabs) - 1
			}
		case "p":
			m.send(ActionTogglePause)
		case "r":
			m.send(ActionRefresh)
		case "?":
			m.help = !m.help
		}
	case snapshotMsg:
		m.snapshot = miner.Snapshot(v)
		return m, waitSnapshot(m.updates)
	case eventMsg:
		m.logs = append(m.logs, miner.Event(v))
		if len(m.logs) > 200 {
			m.logs = m.logs[len(m.logs)-200:]
		}
		return m, waitEvent(m.events)
	}
	return m, nil
}

func (m Model) send(action Action) {
	select {
	case m.actions <- action:
	default:
	}
}

func waitSnapshot(ch <-chan miner.Snapshot) tea.Cmd {
	return func() tea.Msg {
		v := <-ch
		return snapshotMsg(v)
	}
}

func waitEvent(ch <-chan miner.Event) tea.Cmd {
	return func() tea.Msg {
		v := <-ch
		return eventMsg(v)
	}
}

func (m Model) sortedSettingKeys() []string {
	keys := make([]string, 0, len(m.settings))
	for key := range m.settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

