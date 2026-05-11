package tui

type Action int

const (
	ActionNone Action = iota
	ActionQuit
	ActionTogglePause
	ActionRefresh
)

