import Testing

@testable import Todo

@Test("[scenario.todo.toggle.complete] toggling an active todo completes it")
func complete() throws {
	let t = try Todo(label: "x", done: false).toggled()
	#expect(t.done == true)
}

@Test("[scenario.todo.toggle.reactivate] toggling a completed todo reactivates it")
func reactivate() throws {
	let t = try Todo(label: "x", done: true).toggled()
	#expect(t.done == false)
}

@Test("[scenario.todo.toggle.guard-empty] toggling an empty-label todo is rejected")
func guardEmpty() throws {
	// Expects a throw; the impl never throws (intentional bug) -> this FAILS,
	// while a (deviates:) marker in Todo.swift claims it is intentional.
	#expect(throws: TodoError.self) {
		_ = try Todo(label: "", done: false).toggled()
	}
}
