// Package scaffold renders a stack's starter template — a scaffold.json manifest
// plus a files/ tree — into a target directory via Go text/template.
// See docs/design/stack-scaffolding.md.
package scaffold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

// Manifest is a scaffold's scaffold.json.
type Manifest struct {
	Stack     string             `json:"stack"`
	Install   string             `json:"install,omitempty"`
	Target    ManifestTarget     `json:"target"`
	Variables []Variable         `json:"variables,omitempty"`
	Features  map[string]Feature `json:"features,omitempty"`
}

// ManifestTarget holds the specs.json target fields as text/template strings
// (e.g. "{{.Dir}}/junit.xml"), resolved against the Data by RenderTarget.
type ManifestTarget struct {
	Command string `json:"command,omitempty"`
	Format  string `json:"format"`
	Report  string `json:"report"`
	Source  string `json:"source"`
}

// Variable is a manifest-declared template variable (resolved by the CLI).
type Variable struct {
	Name    string `json:"name"`
	Default string `json:"default,omitempty"`
	From    string `json:"from,omitempty"` // "flag" | "git"
}

// Feature is an optional --with add-on: an extra files subtree + vars.
type Feature struct {
	Files string            `json:"files,omitempty"`
	Vars  map[string]string `json:"vars,omitempty"`
}

// Data is the text/template context every scaffold sees.
type Data struct {
	Name     string
	Dir      string
	Product  string
	Vars     map[string]string
	Features map[string]bool
}

// RenderedTarget is a manifest's target after substitution — the specs.json entry.
type RenderedTarget struct{ Command, Format, Report, Source string }

var funcs = template.FuncMap{
	"lower":  strings.ToLower,
	"upper":  strings.ToUpper,
	"kebab":  kebab,
	"pascal": pascal,
	"camel":  camel,
}

// LoadManifest reads scaffold.json from a scaffold's root fs.
func LoadManifest(src fs.FS) (Manifest, error) {
	b, err := fs.ReadFile(src, "scaffold.json")
	if err != nil {
		return Manifest{}, fmt.Errorf("scaffold.json: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("scaffold.json: %w", err)
	}
	return m, nil
}

// Render walks the scaffold's files/ subtree, renders *.tmpl through
// text/template (stripping the suffix) and copies everything else verbatim into
// destDir. Returns the written paths (relative to destDir).
func Render(src fs.FS, destDir string, data Data) ([]string, error) {
	return renderSubtree(src, "files", destDir, data)
}

// renderSubtree renders one subtree (files/ or a feature's dir) into destDir.
func renderSubtree(src fs.FS, root, destDir string, data Data) ([]string, error) {
	var written []string
	err := fs.WalkDir(src, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		if rel == "" {
			return nil
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(destDir, rel), 0o755)
		}
		b, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		out := rel
		if strings.HasSuffix(rel, ".tmpl") {
			s, err := renderString(p, string(b), data)
			if err != nil {
				return err
			}
			b, out = []byte(s), strings.TrimSuffix(rel, ".tmpl")
		}
		dst := filepath.Join(destDir, out)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			return err
		}
		written = append(written, out)
		return nil
	})
	return written, err
}

// RenderFeature renders a feature's files subtree into destDir.
func RenderFeature(src fs.FS, f Feature, destDir string, data Data) ([]string, error) {
	if f.Files == "" {
		return nil, nil
	}
	return renderSubtree(src, f.Files, destDir, data)
}

// RenderTarget resolves the manifest's target fields against data.
func RenderTarget(m Manifest, data Data) (RenderedTarget, error) {
	var rt RenderedTarget
	var err error
	if rt.Command, err = renderString("target.command", m.Target.Command, data); err != nil {
		return rt, err
	}
	if rt.Format, err = renderString("target.format", m.Target.Format, data); err != nil {
		return rt, err
	}
	if rt.Report, err = renderString("target.report", m.Target.Report, data); err != nil {
		return rt, err
	}
	if rt.Source, err = renderString("target.source", m.Target.Source, data); err != nil {
		return rt, err
	}
	return rt, nil
}

func renderString(name, text string, data Data) (string, error) {
	if text == "" {
		return "", nil
	}
	t, err := template.New(name).Funcs(funcs).Parse(text)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return buf.String(), nil
}

// --- casing helpers ---

func tokens(s string) []string {
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	var prev rune
	for i, r := range s {
		switch {
		case r == '-' || r == '_' || r == ' ' || r == '.' || r == '/':
			flush()
		case i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(prev):
			flush()
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
		prev = r
	}
	flush()
	return out
}

func kebab(s string) string {
	parts := tokens(s)
	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}
	return strings.Join(parts, "-")
}

func pascal(s string) string {
	parts := tokens(s)
	for i := range parts {
		parts[i] = title1(parts[i])
	}
	return strings.Join(parts, "")
}

func camel(s string) string {
	p := pascal(s)
	if p == "" {
		return p
	}
	r := []rune(p)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func title1(s string) string {
	if s == "" {
		return s
	}
	r := []rune(strings.ToLower(s))
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
