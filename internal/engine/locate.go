package engine

import (
	"os"

	"github.com/markmals/speckit/internal/specmodel"
)

// SpecLocation is where a scenario is declared in the library — used to point CI
// annotations at the source. File is relative to the library root.
type SpecLocation struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// SpecLocations maps each declared scenario sub-id to where it is declared, so a
// gate failure (an unjoinable scenario, a drifted cell) can be annotated at the
// spec line that declares it. Scenarios with no sub-id (an I6 scan violation)
// are omitted.
//
// SPEC: story.engine.verify, story.engine.parity
func SpecLocations(root string) (map[specmodel.SpecID]SpecLocation, error) {
	specs, err := specmodel.LoadLibrary(os.DirFS(root))
	if err != nil {
		return nil, err
	}
	locs := map[specmodel.SpecID]SpecLocation{}
	for _, s := range specs {
		for _, sc := range s.Scenarios {
			if sc.SubID != "" {
				locs[specmodel.SpecID(sc.SubID)] = SpecLocation{File: s.Path, Line: sc.Line}
			}
		}
	}
	return locs, nil
}
