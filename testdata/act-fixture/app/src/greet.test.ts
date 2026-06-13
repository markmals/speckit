import { expect, it } from "vitest";

import { greet } from "./greet.ts";

it("[scenario.greet.hello] greets a user by name", () => {
    expect(greet("Ada")).toBe("Hello, Ada!");
});
