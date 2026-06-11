package project

// Adapter projects the command set + orientation file into an agent's native
// format and location (D4). Adding an agent is implementing this interface —
// the shared command prompts are unchanged.
//
// SPEC: story.init.projection
type Adapter interface {
	// ID is the integration id, e.g. "claude".
	ID() string
	// Project writes the commands + orientation file under root, returning the
	// paths it wrote.
	Project(root string, commands []Command) ([]string, error)
}

var adapters = map[string]Adapter{}

func register(a Adapter) { adapters[a.ID()] = a }

// AdapterFor returns the adapter for an integration id.
func AdapterFor(id string) (Adapter, bool) {
	a, ok := adapters[id]
	return a, ok
}

// AdapterIDs lists the registered integration ids.
func AdapterIDs() []string {
	ids := make([]string, 0, len(adapters))
	for id := range adapters {
		ids = append(ids, id)
	}
	return ids
}
