// Package markdown is the default work provider: one committed markdown
// file whose `##` sections are the state machine. No network, no external
// binary, no dependency graph.
package markdown

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/markmals/speckit/internal/work"
)

// Provider reads and rewrites one committed work file.
type Provider struct {
	path string
}

var _ work.Provider = (*Provider)(nil)

// New returns the provider on file under root (file may be absolute).
func New(root, file string) *Provider {
	if !filepath.IsAbs(file) {
		file = filepath.Join(root, file)
	}
	return &Provider{path: file}
}

func (p *Provider) Name() string { return "markdown" }

func (p *Provider) Ready(ctx context.Context) ([]work.Item, error) {
	return p.List(ctx, work.StateReady)
}

// List returns a state's items, or every item for "". A missing file is an
// empty board. Any state name is valid — an unknown one is just empty.
func (p *Provider) List(_ context.Context, state string) ([]work.Item, error) {
	b, err := p.load()
	if err != nil {
		return nil, err
	}
	return b.list(work.Slug(state)), nil
}

// Create allocates the next wk-<n> id and lands the item in ready,
// creating the file if it doesn't exist yet.
func (p *Provider) Create(_ context.Context, req work.CreateRequest) (work.Item, error) {
	if strings.TrimSpace(req.Title) == "" {
		return work.Item{}, fmt.Errorf("create: a title is required")
	}
	b, err := p.load()
	if err != nil {
		return work.Item{}, err
	}
	typ := req.Type
	if typ == work.TypeTask { // "" == task on an Item
		typ = ""
	}
	it := b.add(work.StateReady, work.Item{ID: b.nextID(), Title: req.Title, Type: typ, Spec: req.Spec})
	if err := p.save(b); err != nil {
		return work.Item{}, err
	}
	return it, nil
}

func (p *Provider) Claim(_ context.Context, id string) (work.Item, error) {
	return p.move(id, work.StateInProgress)
}

// Move accepts ANY state — a state with no section gets one.
func (p *Provider) Move(_ context.Context, id, state string) (work.Item, error) {
	s := work.Slug(state)
	if s == "" {
		return work.Item{}, fmt.Errorf("move: %q is not a state", state)
	}
	return p.move(id, s)
}

func (p *Provider) move(id, state string) (work.Item, error) {
	b, err := p.load()
	if err != nil {
		return work.Item{}, err
	}
	it, ok := b.take(id)
	if !ok {
		return work.Item{}, fmt.Errorf("no item %q in %s", id, p.path)
	}
	it = b.add(state, it)
	if err := p.save(b); err != nil {
		return work.Item{}, err
	}
	return it, nil
}

func (p *Provider) load() (board, error) {
	src, err := os.ReadFile(p.path)
	if errors.Is(err, fs.ErrNotExist) {
		return parse(""), nil // a missing file is an empty board
	}
	if err != nil {
		return board{}, err
	}
	return parse(string(src)), nil
}

// save writes atomically: a temp file in the same directory, then rename.
func (p *Provider) save(b board) error {
	dir := filepath.Dir(p.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(p.path)+".*")
	if err != nil {
		return err
	}
	_, werr := tmp.WriteString(render(b))
	if err := tmp.Close(); werr == nil {
		werr = err
	}
	if werr == nil {
		werr = os.Chmod(tmp.Name(), 0o644)
	}
	if werr == nil {
		werr = os.Rename(tmp.Name(), p.path)
	}
	if werr != nil {
		os.Remove(tmp.Name())
	}
	return werr
}
