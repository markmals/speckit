import Testing

// Test-association traits, shared across every test target via `import TestSupport`.
// `.spec(_:)` binds a suite (or test) to the spec ID it verifies; `.scenario(_:)`
// binds a test to the Gherkin scenario sub-ID it pins. Dotted IDs are carried
// verbatim in source so `specify verify` reads the binding straight from the trait
// — never from the test's name. Public so each test target can import them.

public struct SpecTrait: TestTrait, SuiteTrait {
    public let id: String
}

public struct ScenarioTrait: TestTrait {
    public let id: String
}

extension Trait where Self == SpecTrait {
    public static func spec(_ id: String) -> Self { SpecTrait(id: id) }
}

extension Trait where Self == ScenarioTrait {
    public static func scenario(_ id: String) -> Self { ScenarioTrait(id: id) }
}
