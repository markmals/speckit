package markdown

import (
	"strconv"
	"strings"

	"github.com/markmals/speckit/internal/work"
)

// board is a parsed work file: the preamble (everything before the first
// `##`, kept verbatim minus trailing blank lines) plus items grouped by
// state in first-seen order. The `##` sections ARE the state machine.
type board struct {
	preamble string
	order    []string // states in first-seen order
	items    map[string][]work.Item
}

// defaultPreamble heads a file that had none.
const defaultPreamble = "# Work\n\nMaintained by `specify work` — items move between sections as their state changes."

// parse reads a work file. Any `## Heading` opens a state named
// work.Slug(heading); lines inside a section that don't parse as items are
// dropped on the next rewrite.
func parse(src string) board {
	b := board{items: map[string][]work.Item{}}
	var pre []string
	state := ""
	for _, line := range strings.Split(src, "\n") {
		if h, ok := strings.CutPrefix(line, "## "); ok {
			state = work.Slug(h)
			b.section(state)
			continue
		}
		if state == "" {
			pre = append(pre, line)
			continue
		}
		if it, ok := parseItem(line, state); ok {
			b.items[state] = append(b.items[state], it)
		}
	}
	b.preamble = strings.TrimRight(strings.Join(pre, "\n"), "\n")
	return b
}

// parseItem reads one item line: `- [ ] ` or `- [x] `, the backticked id,
// the title, then optional ` · `-separated trailing fields (`spec: <id>`,
// `type: defect`). Fields are recognized from the right, so a title
// containing " · " survives. The checkbox is presentation only — state
// comes from the section.
func parseItem(line, state string) (work.Item, bool) {
	rest, ok := strings.CutPrefix(line, "- [ ] ")
	if !ok {
		rest, ok = strings.CutPrefix(line, "- [x] ")
	}
	if !ok {
		return work.Item{}, false
	}
	rest, ok = strings.CutPrefix(strings.TrimSpace(rest), "`")
	if !ok {
		return work.Item{}, false
	}
	id, rest, ok := strings.Cut(rest, "`")
	if !ok || id == "" {
		return work.Item{}, false
	}
	it := work.Item{ID: id, State: state}
	parts := strings.Split(strings.TrimSpace(rest), " · ")
	n := len(parts)
	for n > 1 {
		if v, ok := strings.CutPrefix(parts[n-1], "spec: "); ok {
			it.Spec = v
			n--
			continue
		}
		if v, ok := strings.CutPrefix(parts[n-1], "type: "); ok {
			if v != work.TypeTask { // "" == task on an Item
				it.Type = v
			}
			n--
			continue
		}
		break
	}
	it.Title = strings.Join(parts[:n], " · ")
	if it.Title == "" {
		return work.Item{}, false
	}
	return it, true
}

// render writes the board deterministically: preamble, the canonical states
// in canonical order (empty ones included — they are the documented
// vocabulary), then any other state in first-seen order (empty ones
// dropped). Done items render `- [x]`; every other state `- [ ]`.
func render(b board) string {
	var sb strings.Builder
	pre := b.preamble
	if pre == "" {
		pre = defaultPreamble
	}
	sb.WriteString(pre)
	sb.WriteString("\n")
	writeSection := func(state string) {
		sb.WriteString("\n## ")
		sb.WriteString(work.Heading(state))
		sb.WriteString("\n")
		if len(b.items[state]) == 0 {
			return
		}
		sb.WriteString("\n")
		for _, it := range b.items[state] {
			sb.WriteString(itemLine(it))
			sb.WriteString("\n")
		}
	}
	for _, state := range work.CanonicalStates {
		writeSection(state)
	}
	for _, state := range b.order {
		if !work.IsCanonical(state) && len(b.items[state]) > 0 {
			writeSection(state)
		}
	}
	return sb.String()
}

func itemLine(it work.Item) string {
	box := "- [ ] "
	if it.State == work.StateDone {
		box = "- [x] "
	}
	line := box + "`" + it.ID + "` " + it.Title
	if it.Spec != "" {
		line += " · spec: " + it.Spec
	}
	if it.Type != "" {
		line += " · type: " + it.Type
	}
	return line
}

// section registers a state the first time it is seen.
func (b *board) section(state string) {
	if _, ok := b.items[state]; !ok {
		b.order = append(b.order, state)
		b.items[state] = nil
	}
}

// add appends the item to a state's section, creating the section if new.
func (b *board) add(state string, it work.Item) work.Item {
	it.State = state
	b.section(state)
	b.items[state] = append(b.items[state], it)
	return it
}

// take removes and returns the item by id, wherever it sits.
func (b *board) take(id string) (work.Item, bool) {
	for state, items := range b.items {
		for i, it := range items {
			if it.ID == id {
				b.items[state] = append(items[:i:i], items[i+1:]...)
				return it, true
			}
		}
	}
	return work.Item{}, false
}

// list returns a state's items, or every item in render order for "".
func (b board) list(state string) []work.Item {
	if state != "" {
		return b.items[state]
	}
	var out []work.Item
	for _, s := range work.CanonicalStates {
		out = append(out, b.items[s]...)
	}
	for _, s := range b.order {
		if !work.IsCanonical(s) {
			out = append(out, b.items[s]...)
		}
	}
	return out
}

// nextID allocates max(existing wk-<n>) + 1, so an id is never reused even
// after a delete leaves a gap.
func (b board) nextID() string {
	max := 0
	for _, items := range b.items {
		for _, it := range items {
			if n, ok := strings.CutPrefix(it.ID, "wk-"); ok {
				if v, err := strconv.Atoi(n); err == nil && v > max {
					max = v
				}
			}
		}
	}
	return "wk-" + strconv.Itoa(max+1)
}
