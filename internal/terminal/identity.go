package terminal

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
	"github.com/raywcm/taskdock/internal/i18n"
	"github.com/raywcm/taskdock/internal/task"
)

// Save/restore the window title without querying the terminal or consuming
// application input. Supported terminals restore the title on normal detach.
const saveWindowTitle = "\x1b[22;2t"
const restoreWindowTitle = "\x1b[23;2t"

type identity struct {
	title, line string
}

func newIdentity(spec task.Spec, locale i18n.Locale, size task.Size, noEscape bool) identity {
	// Metadata is validated at the service boundary. Also remove controls here
	// so a future caller cannot inject OSC terminators or terminal commands.
	name := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, spec.Name)
	id := string(spec.ID)
	if len(id) > 8 {
		id = id[:8]
	}
	title := fmt.Sprintf("TaskDock · %s [%s]", name, id)
	width := max(1, int(size.Cols)-1)
	label := title
	if ansi.StringWidth(label) > width {
		prefix, suffix := "TaskDock · ", " ["+id+"]"
		nameWidth := max(1, width-ansi.StringWidth(prefix+suffix))
		label = ansi.Truncate(prefix+ansi.Truncate(name, nameWidth, "…")+suffix, width, "…")
	}
	hint := locale.Text("attach_identity_hint")
	if noEscape {
		hint = locale.Text("attach_no_escape_hint")
	}
	if ansi.StringWidth(label+" · "+hint) <= width {
		label += " · " + hint
	}
	return identity{title: title, line: label}
}

func (i identity) start() string { return saveWindowTitle + ansi.SetWindowTitle(i.title) }

func (i identity) withMode(label string, size task.Size) identity {
	width := max(1, int(size.Cols)-1)
	i.line = ansi.Truncate(i.title, max(1, width-ansi.StringWidth(label)-3), "…") + " · " + label
	i.line = ansi.Truncate(i.line, width, "…")
	i.title += " · " + label
	return i
}
func (i identity) banner() string {
	return "\r\n" + i.line + "\r\n"
}
