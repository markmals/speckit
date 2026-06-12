// Package specmodel is the mechanized form of specs/CONVENTIONS.md: the
// frontmatter schema, the closed kind taxonomy, dotted stable IDs, scenario
// sub-IDs, reverse pointers, and deviation markers.
//
// Per the fork plan (§8) this is the package the core binary and the spec
// engine share, so it lands first. Phase 0 establishes the types and the
// closed kind set; the frontmatter/scenario parser arrives in Phase 3.
//
// SPEC: domain.specmodel
package specmodel

// Kind is one of the closed set of allowed `kind:` values from CONVENTIONS.md.
type Kind string

const (
	KindNarrative    Kind = "narrative"
	KindStory        Kind = "story"
	KindUseCase      Kind = "use-case"
	KindFlow         Kind = "flow"
	KindDomain       Kind = "domain"
	KindViewModel    Kind = "view-model"
	KindError        Kind = "error"
	KindArchitecture Kind = "architecture"
	KindDesignSystem Kind = "design-system"
	KindConventions  Kind = "conventions"
)

// Kinds is the closed taxonomy. Adding a kind is a deliberate change to
// CONVENTIONS.md, not an ad-hoc choice — so this list is the single source of
// truth the scanner validates `kind:` frontmatter against.
var Kinds = []Kind{
	KindNarrative, KindStory, KindUseCase, KindFlow, KindDomain,
	KindViewModel, KindError, KindArchitecture, KindDesignSystem, KindConventions,
}

// Prefix is the dotted ID prefix a kind's IDs must start with (CONVENTIONS.md
// kind taxonomy). Empty for the singular cross-cutting kinds whose ID is just
// the kind name (architecture, design-system, conventions).
func (k Kind) Prefix() string {
	switch k {
	case KindStory:
		return "story."
	case KindUseCase:
		return "usecase."
	case KindFlow:
		return "flow."
	case KindDomain:
		return "domain."
	case KindViewModel:
		return "vm."
	case KindError:
		return "error."
	case KindNarrative:
		return "narrative."
	default:
		return "" // architecture | design-system | conventions
	}
}

// Valid reports whether k is a member of the closed taxonomy.
func (k Kind) Valid() bool {
	for _, known := range Kinds {
		if k == known {
			return true
		}
	}
	return false
}

// Singular reports whether the kind is a singular cross-cutting file (one per
// product or feature, with a special uppercase filename like CONVENTIONS.md or
// NARRATIVE.md) rather than one-per-file in a directory — so it is exempt from
// the filename↔id rule (I1).
func (k Kind) Singular() bool {
	switch k {
	case KindNarrative, KindArchitecture, KindDesignSystem, KindConventions:
		return true
	}
	return false
}

// SpecID is a dotted, lowercase, hierarchical, stable identifier — e.g.
// domain.item, vm.items.list, story.item.create. IDs are immutable once an
// implementation references them; they do not encode target.
type SpecID string

// Frontmatter is the YAML header every spec file carries. status defaults to
// "accepted" when omitted.
type Frontmatter struct {
	ID        SpecID   `yaml:"id"`
	Kind      Kind     `yaml:"kind"`
	DependsOn []SpecID `yaml:"depends-on,omitempty"`
	Status    string   `yaml:"status,omitempty"`
}
