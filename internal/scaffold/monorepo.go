package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2/unstable"
)

// span is a half-open byte range [start,end) into a TOML document.
type span struct{ start, end int }

// exprKind classifies a parsed top-level expression. We only branch on a few
// kinds, so we wrap unstable.Kind with the predicates we need.
type exprKind unstable.Kind

func (k exprKind) isTable() bool {
	return unstable.Kind(k) == unstable.Table || unstable.Kind(k) == unstable.ArrayTable
}
func (k exprKind) isKeyValue() bool { return unstable.Kind(k) == unstable.KeyValue }

// expr is one top-level parsed expression with a real source span. For a table,
// name is its dotted key joined by "." (e.g. tasks.fmt:check); span is the
// derived [..] header line. For a key/value, name is the key, span is the whole
// "key = value", and val is the decoded string value (empty for non-strings).
type expr struct {
	kind exprKind
	span span
	name string
	val  string
}

// parseExprs returns every top-level expression with its source span. Tables get
// a derived header span (the unstable parser gives table nodes a zero Raw).
func parseExprs(data []byte) ([]expr, error) {
	p := &unstable.Parser{KeepComments: true}
	p.Reset(data)
	var out []expr
	for p.NextExpression() {
		e := p.Expression()
		switch e.Kind {
		case unstable.Table, unstable.ArrayTable:
			keyOff, keyEnd := -1, -1
			var parts []string
			it := e.Key()
			for it.Next() {
				k := it.Node()
				parts = append(parts, string(k.Data))
				if keyOff == -1 {
					keyOff = int(k.Raw.Offset)
				}
				keyEnd = int(k.Raw.Offset + k.Raw.Length)
			}
			hs := bytes.LastIndexByte(data[:keyOff], '[')
			// An array-table header opens with "[[" — step back over the second
			// bracket so the span covers the whole "[[name]]" (we never emit these
			// today, but isTable() admits ArrayTable, so keep the span honest).
			if e.Kind == unstable.ArrayTable && hs > 0 && data[hs-1] == '[' {
				hs--
			}
			he := len(data)
			if nl := bytes.IndexByte(data[keyEnd:], '\n'); nl >= 0 {
				he = keyEnd + nl
			}
			out = append(out, expr{exprKind(e.Kind), span{hs, he}, strings.Join(parts, "."), ""})
		case unstable.KeyValue:
			var parts []string
			it := e.Key()
			for it.Next() {
				parts = append(parts, string(it.Node().Data))
			}
			val := ""
			if v := e.Value(); v != nil {
				val = string(v.Data)
			}
			out = append(out, expr{exprKind(e.Kind), span{int(e.Raw.Offset), int(e.Raw.Offset + e.Raw.Length)}, strings.Join(parts, "."), val})
		default:
			out = append(out, expr{exprKind(e.Kind), span{int(e.Raw.Offset), int(e.Raw.Offset + e.Raw.Length)}, "", ""})
		}
	}
	return out, p.Error()
}

// sectionEnd returns the byte offset just after the last key/value belonging to
// the table whose header is exprs[i] — i.e. the insertion point for a new key,
// before the next table header or EOF.
func sectionEnd(exprs []expr, i int) int {
	end := exprs[i].span.end
	for j := i + 1; j < len(exprs); j++ {
		if exprs[j].kind.isTable() {
			break
		}
		if exprs[j].kind.isKeyValue() && exprs[j].span.end > end {
			end = exprs[j].span.end
		}
	}
	return end
}

// splice returns data with ins inserted at byte offset at.
func splice(data []byte, at int, ins string) []byte {
	out := make([]byte, 0, len(data)+len(ins))
	out = append(out, data[:at]...)
	out = append(out, ins...)
	out = append(out, data[at:]...)
	return out
}

var varsRe = regexp.MustCompile(`\{\{\s*vars\.(\w+)\s*\}\}`)

// substituteVars resolves mise's {{ vars.X }} interpolations against vars — used
// to compute a family template's canonical run for a given member before the
// promotion equality check.
func substituteVars(s string, vars map[string]string) string {
	return varsRe.ReplaceAllStringFunc(s, func(m string) string {
		name := varsRe.FindStringSubmatch(m)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return m
	})
}

// ToolPin is one [tools] entry from a family contribution, kept ordered.
type ToolPin struct{ Key, Val string }

// Template is one [task_templates] body: the run string (possibly with mise
// {{ vars.X }} interpolations) and an optional description.
type Template struct{ Run, Description string }

// Family is a stack family's mise contribution, parsed from
// templates/monorepo/<name>.toml: its toolchain pins, its task templates (keyed
// by the bare task name, i.e. without the "<family>:" prefix), and the verbatim
// [task_templates.*] block text for EOF append. Hoist is set by the caller when
// the family has reached two members (so EnsureRootMise appends Raw).
type Family struct {
	Name      string
	Tools     []ToolPin
	Templates map[string]Template
	Raw       string
	Hoist     bool
}

// LoadFamily reads templates/monorepo/<name>.toml from the assets FS.
func LoadFamily(assets fs.FS, name string) (Family, error) {
	data, err := fs.ReadFile(assets, "templates/monorepo/"+name+".toml")
	if err != nil {
		return Family{}, fmt.Errorf("family %q: %w", name, err)
	}
	fam := Family{Name: name, Templates: map[string]Template{}}

	ex, err := parseExprs(data)
	if err != nil {
		return Family{}, fmt.Errorf("family %q: %w", name, err)
	}
	// Walk sections: collect [tools] pins (ordered) and each [task_templates."fam:task"].
	rawStart := -1
	for i, e := range ex {
		if !e.kind.isTable() {
			continue
		}
		switch {
		case e.name == "tools":
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() {
					fam.Tools = append(fam.Tools, ToolPin{ex[j].name, ex[j].val})
				}
			}
		case strings.HasPrefix(e.name, "task_templates."):
			if rawStart == -1 {
				rawStart = e.span.start
			}
			// the template's dotted name part after task_templates. e.g. node:test
			full := strings.TrimPrefix(e.name, "task_templates.")
			task := full
			if c := strings.IndexByte(full, ':'); c >= 0 {
				task = full[c+1:] // strip the "<family>:" prefix
			}
			tpl := Template{}
			for j := i + 1; j < len(ex); j++ {
				if ex[j].kind.isTable() {
					break
				}
				if ex[j].kind.isKeyValue() {
					switch ex[j].name {
					case "run":
						tpl.Run = ex[j].val
					case "description":
						tpl.Description = ex[j].val
					}
				}
			}
			fam.Templates[task] = tpl
		}
	}
	if rawStart >= 0 {
		fam.Raw = strings.TrimRight(string(data[rawStart:]), "\n") + "\n"
	}
	return fam, nil
}

// EnsureRootMise creates or merges the repo-root mise.toml so it declares
// monorepo_root, the config_roots globs for every target dir, and the union of
// the present families' [tools]. A family's [task_templates] are appended only
// when that family is marked Hoist (it has reached two members). Idempotent:
// re-running adds nothing. Preserves all existing comments and user content via
// surgical byte-range splices. Reports whether it changed the file.
func EnsureRootMise(root string, families []Family, targetDirs []string) (bool, error) {
	misePath := filepath.Join(root, "mise.toml")
	orig, err := os.ReadFile(misePath)
	if os.IsNotExist(err) {
		orig = nil
	} else if err != nil {
		return false, err
	}

	data := orig
	if len(data) == 0 {
		data = []byte(rootSkeleton(families, targetDirs))
	} else {
		if data, err = ensureMonorepoRoot(data); err != nil {
			return false, err
		}
		for _, g := range globsFor(targetDirs) {
			if data, err = ensureConfigRoot(data, g); err != nil {
				return false, err
			}
		}
		for _, fam := range families {
			if data, err = ensureTools(data, fam.Tools); err != nil {
				return false, err
			}
		}
	}
	// Append any hoisted family's templates that aren't present yet.
	for _, fam := range families {
		if !fam.Hoist || fam.Raw == "" {
			continue
		}
		if !bytes.Contains(data, []byte(`[task_templates."`+fam.Name+`:`)) {
			if !bytes.HasSuffix(data, []byte("\n")) {
				data = append(data, '\n')
			}
			data = append(data, '\n')
			data = append(data, fam.Raw...)
		}
	}

	if bytes.Equal(data, orig) {
		return false, nil
	}
	if err := os.WriteFile(misePath, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// rootSkeleton renders a fresh managed root config.
func rootSkeleton(families []Family, targetDirs []string) string {
	var b strings.Builder
	b.WriteString("# Managed by `specify target` — your edits and comments are preserved.\n")
	b.WriteString("# Task bodies live in each member's mise.toml until a family has two members,\n")
	b.WriteString("# then move to a task_templates table here.\n")
	b.WriteString("monorepo_root = true\n\n")
	b.WriteString("[monorepo]\n")
	globs := globsFor(targetDirs)
	quoted := make([]string, len(globs))
	for i, g := range globs {
		quoted[i] = `"` + g + `"`
	}
	b.WriteString("config_roots = [" + strings.Join(quoted, ", ") + "]\n")
	// Union of family tools, in family order then pin order.
	var pins []ToolPin
	seen := map[string]bool{}
	for _, fam := range families {
		for _, p := range fam.Tools {
			if !seen[p.Key] {
				seen[p.Key] = true
				pins = append(pins, p)
			}
		}
	}
	if len(pins) > 0 {
		b.WriteString("\n[tools]\n")
		for _, p := range pins {
			b.WriteString(p.Key + " = \"" + p.Val + "\"\n")
		}
	}
	return b.String()
}

// globsFor maps target dirs to their covering config_roots globs (parent + "/*"),
// deduped and sorted. A dir at the repo root falls back to an explicit entry.
func globsFor(dirs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, d := range dirs {
		d = filepath.ToSlash(d)
		parent := path.Dir(d)
		g := parent + "/*"
		if parent == "." || parent == "" {
			g = d // repo-root member: name it explicitly
		}
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	sort.Strings(out)
	return out
}

// ensureMonorepoRoot splices `monorepo_root = true` after any leading comments
// if the key is absent.
func ensureMonorepoRoot(data []byte) ([]byte, error) {
	ex, err := parseExprs(data)
	if err != nil {
		return nil, err
	}
	for _, e := range ex {
		if e.kind.isKeyValue() && e.name == "monorepo_root" {
			return data, nil
		}
	}
	// insert at the first non-comment position (or EOF preamble).
	at := 0
	for _, e := range ex {
		if unstable.Kind(e.kind) == unstable.Comment {
			if e.span.end > at {
				at = e.span.end
			}
			continue
		}
		break
	}
	ins := "monorepo_root = true\n"
	if at > 0 {
		ins = "\n" + ins
	}
	return splice(data, at, ins), nil
}

// ensureConfigRoot ensures glob is an element of [monorepo].config_roots,
// splicing it before the array's closing ] when missing. Creates the [monorepo]
// table + key if absent.
func ensureConfigRoot(data []byte, glob string) ([]byte, error) {
	ex, err := parseExprs(data)
	if err != nil {
		return nil, err
	}
	for _, e := range ex {
		if e.kind.isKeyValue() && e.name == "config_roots" {
			seg := data[e.span.start:e.span.end]
			if bytes.Contains(seg, []byte(`"`+glob+`"`)) {
				return data, nil
			}
			rb := bytes.LastIndexByte(seg, ']')
			// handle empty array "[]" vs "[ ... ]"
			inner := bytes.TrimSpace(seg[bytes.IndexByte(seg, '[')+1 : rb])
			ins := `, "` + glob + `"`
			if len(inner) == 0 {
				ins = `"` + glob + `"`
			}
			return splice(data, e.span.start+rb, ins), nil
		}
	}
	// No config_roots key — ensure a [monorepo] table holds one.
	for i, e := range ex {
		if e.kind.isTable() && e.name == "monorepo" {
			at := sectionEnd(ex, i)
			return splice(data, at, "\nconfig_roots = [\""+glob+"\"]"), nil
		}
	}
	// No [monorepo] table at all — append one.
	out := data
	if !bytes.HasSuffix(out, []byte("\n")) {
		out = append(out, '\n')
	}
	return append(out, []byte("\n[monorepo]\nconfig_roots = [\""+glob+"\"]\n")...), nil
}

// ensureTools splices each missing pin after [tools]'s last key (never
// overwriting a user-pinned version). Creates [tools] if absent.
func ensureTools(data []byte, pins []ToolPin) ([]byte, error) {
	for _, pin := range pins {
		ex, err := parseExprs(data)
		if err != nil {
			return nil, err
		}
		toolsIdx := -1
		present := false
		for i, e := range ex {
			if e.kind.isTable() && e.name == "tools" {
				toolsIdx = i
				for j := i + 1; j < len(ex); j++ {
					if ex[j].kind.isTable() {
						break
					}
					if ex[j].kind.isKeyValue() && ex[j].name == pin.Key {
						present = true
					}
				}
			}
		}
		if present {
			continue
		}
		if toolsIdx == -1 {
			out := data
			if !bytes.HasSuffix(out, []byte("\n")) {
				out = append(out, '\n')
			}
			data = append(out, []byte("\n[tools]\n"+pin.Key+" = \""+pin.Val+"\"\n")...)
			continue
		}
		at := sectionEnd(ex, toolsIdx)
		data = splice(data, at, "\n"+pin.Key+" = \""+pin.Val+"\"")
	}
	return data, nil
}
