import { describe, expect, it } from "vitest";
import { toggle } from "../src/todo";

describe("story.todo.toggle", () => {
	it("[scenario.todo.toggle.complete] toggling an active todo completes it", () => {
		expect(toggle({ label: "x", done: false }).done).toBe(true);
	});

	it("[scenario.todo.toggle.reactivate] toggling a completed todo reactivates it", () => {
		expect(toggle({ label: "x", done: true }).done).toBe(false);
	});

	it("[scenario.todo.toggle.guard-empty] toggling an empty-label todo is rejected", () => {
		expect(() => toggle({ label: "", done: false })).toThrow();
	});

	// D12: this tag references a scenario the spec does not declare (renamed/typo).
	// The test itself passes — the dishonesty is in the join, which the engine must catch.
	it("[scenario.todo.toggle.complete-typo] dangling reference demo", () => {
		expect(toggle({ label: "y", done: false }).done).toBe(true);
	});
});
