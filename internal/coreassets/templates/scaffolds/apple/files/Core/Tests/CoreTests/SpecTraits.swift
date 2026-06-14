import Testing

// Test-association traits. `.spec(_:)` binds a suite (or test) to the spec ID it
// verifies; `.scenario(_:)` binds a test to the Gherkin scenario sub-ID it pins.
// Dotted IDs are carried verbatim in source so `specify verify` reads the binding
// straight from the trait — never from the test's name.

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
