package tui

import tea "charm.land/bubbletea/v2"

type checkedName struct {
	kind, name, suggested string
	err                   error
}

func (m model) submitName(kind, name string) (tea.Model, tea.Cmd) {
	m.nameKind = kind
	if kind == "new" {
		m.selected = Selection{Kind: "new", Name: name, State: m.state()}
		return m, tea.Quit
	}
	id := m.target
	m.mode = ""
	m.busy = true
	m.input.SetValue(name)
	return m, func() tea.Msg { return changed{m.catalog.Rename(m.ctx, string(id), name)} }
}

// Confirmation defaults to Cancel. Only y, or selecting the action and pressing
// Enter, accepts it. Typing other unrelated keys cancels without side effects.
func (m *model) confirmKey(key string) (yes, done bool) {
	switch key {
	case "left", "right", "tab", "shift+tab":
		m.confirmChoice = 1 - m.confirmChoice
		return false, false
	case "y":
		return true, true
	case "enter":
		return m.confirmChoice == 1, true
	default:
		return false, true
	}
}
