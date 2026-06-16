package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"regexp"
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
