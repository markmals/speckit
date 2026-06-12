import { expect, it } from "vitest";

import { greeting } from "./greeting.ts";

// The scenario binding is the it() title's [scenario.<id>] prefix (CONVENTIONS).
it("[scenario.welcome.greet.hello] greets a user by name", () => {
    expect(greeting("Ada")).toBe("Hello, Ada!");
});
