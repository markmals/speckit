import Testing

// SPEC: manual
// Test-association traits (mirrors Reactivity's SpecTraits.swift). `.spec(_:)`
// binds a suite/test to the spec ID it verifies; `.scenario(_:)` binds a test
// to the Gherkin scenario sub-ID it pins. Dotted IDs are carried verbatim so
// the engine can read the binding from source.

struct SpecTrait: TestTrait, SuiteTrait {
	let id: String
}

struct ScenarioTrait: TestTrait {
	let id: String
}

extension Trait where Self == SpecTrait {
	static func spec(_ id: String) -> Self { SpecTrait(id: id) }
}

extension Trait where Self == ScenarioTrait {
	static func scenario(_ id: String) -> Self { ScenarioTrait(id: id) }
}
