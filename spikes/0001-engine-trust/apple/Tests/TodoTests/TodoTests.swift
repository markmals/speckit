import Testing

@testable import Todo

// Convention (from Reactivity): suite bound to the spec via `.spec`, each test
// bound to its scenario via `.scenario`, function names are descriptive raw
// identifiers. The scenario ID lives in the trait (source), NOT the test name.
@Suite(.spec("story.todo.toggle"))
struct TodoToggleTests {
	@Test(.scenario("scenario.todo.toggle.complete"))
	func `toggling an active todo completes it`() throws {
		let t = try Todo(label: "x", done: false).toggled()
		#expect(t.done == true)
	}

	@Test(.scenario("scenario.todo.toggle.reactivate"))
	func `toggling a completed todo reactivates it`() throws {
		let t = try Todo(label: "x", done: true).toggled()
		#expect(t.done == false)
	}

	@Test(.scenario("scenario.todo.toggle.guard-empty"))
	func `toggling an empty-label todo is rejected`() throws {
		// Impl never throws (intentional bug) -> FAILS, under a lying (deviates:) marker.
		#expect(throws: TodoError.self) {
			_ = try Todo(label: "", done: false).toggled()
		}
	}
}
