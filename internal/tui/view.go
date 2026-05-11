package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	tabStyle    = lipgloss.NewStyle().Padding(0, 1)
	activeTab   = tabStyle.Copy().Bold(true).Foreground(lipgloss.Color("10"))
	panel       = lipgloss.NewStyle().Padding(1, 2).Border(lipgloss.RoundedBorder())
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
)

func (m Model) View() string {
	var tabs []string
	for i, t := range m.tabs {
		if i == m.active {
			tabs = append(tabs, activeTab.Render(t))
		} else {
			tabs = append(tabs, tabStyle.Render(t))
		}
	}

	help := "h/l or arrows: tabs | p: pause | r: refresh | ?: help | q: quit"
	if m.help {
		help = "HASH256 GPU miner TUI. Settings come from .env. PRIVATE_KEY is redacted in UI and logs."
	}

	return strings.Join(tabs, "") + "\n" + panel.Render(m.page()) + "\n" + statusStyle.Render(help)
}

func (m Model) page() string {
	s := m.snapshot
	switch m.tabs[m.active] {
	case "Dashboard":
		mode := okStyle.Render("RUNNING")
		if !s.Running {
			mode = warnStyle.Render("PAUSED")
		}
		return fmt.Sprintf(
			"Status: %s\nDry run: %v\nHashrate: %.2f H/s\nAccepted: %d\nRejected: %d\nWallet: %s\nBalance wei: %s\nGPU: %s",
			mode, s.DryRun, s.Hashrate, s.Accepted, s.Rejected, empty(s.Wallet), empty(s.BalanceWei), empty(s.GPU),
		)
	case "Mining":
		return fmt.Sprintf(
			"Nonce range: %d - %d\nTotal hashes: %d\nLast candidate: %d\nLast submit: %s",
			s.LastNonceStart, s.LastNonceEnd, s.TotalHashes, s.LastCandidate, empty(s.LastTX),
		)
	case "Chain":
		return fmt.Sprintf(
			"Block: %s\nEpoch: %s\nEpoch blocks left: %s\nMints in block: %s\nReward wei: %s\nDifficulty: %s\nMinted: %s\nRemaining: %s\nLast refresh: %s",
			empty(s.BlockNumber), empty(s.Epoch), empty(s.EpochBlocksLeft), empty(s.MintsInBlock), empty(s.Reward),
			empty(s.Difficulty), empty(s.Minted), empty(s.Remaining), s.LastStateRefresh.Format("15:04:05"),
		)
	case "Transactions":
		return fmt.Sprintf(
			"Last tx: %s\nLast candidate nonce: %d\nLast submit attempt: %s\nAccepted: %d\nRejected: %d",
			empty(s.LastTX), s.LastCandidate, s.LastSubmitAttempt.Format("15:04:05"), s.Accepted, s.Rejected,
		)
	case "Logs":
		start := 0
		if len(m.logs) > 22 {
			start = len(m.logs) - 22
		}
		lines := []string{}
		for _, entry := range m.logs[start:] {
			lines = append(lines, fmt.Sprintf("%s [%s] %s", entry.At.Format("15:04:05"), entry.Level, entry.Message))
		}
		if len(lines) == 0 {
			return "No logs yet."
		}
		return strings.Join(lines, "\n")
	case "Settings":
		lines := []string{}
		for _, key := range m.sortedSettingKeys() {
			lines = append(lines, fmt.Sprintf("%s=%s", key, m.settings[key]))
		}
		return strings.Join(lines, "\n")
	default:
		return ""
	}
}

func empty(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

