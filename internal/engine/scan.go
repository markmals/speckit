// Package engine implements the spec-engine commands — scan/verify/drift/
// cover/parity/gate/lock/ledger (D2) — over the spec library, built on
// internal/specmodel.
package engine

import (
	"io/fs"

	"github.com/markmals/speckit/internal/specmodel"
)

// Scan loads the spec library from fsys (its specs/ and features/ trees) and
// lints it against the domain.specmodel invariants, returning the findings.
//
// SPEC: story.engine.scan
func Scan(fsys fs.FS) ([]specmodel.Finding, error) {
	specs, err := specmodel.LoadLibrary(fsys)
	if err != nil {
		return nil, err
	}
	return specmodel.Lint(specs), nil
}
