public struct Todo {
	public var label: String
	public var done: Bool
	public init(label: String, done: Bool) {
		self.label = label
		self.done = done
	}
}

public enum TodoError: Error { case emptyLabel }

public extension Todo {
	// SPEC: scenario.todo.toggle.complete
	// SPEC: scenario.todo.toggle.reactivate (deviates: iOS reactivates via long-press)
	// SPEC: scenario.todo.toggle.guard-empty (deviates: native form prevents empty entry)
	//
	// The guard-empty marker above is a LYING deviation: it claims the empty-label
	// guard is intentionally handled natively, but the implementation below never
	// guards, so the guard-empty test FAILS. This is the D11 adversarial case.
	func toggled() throws -> Todo {
		Todo(label: label, done: !done)
	}
}
