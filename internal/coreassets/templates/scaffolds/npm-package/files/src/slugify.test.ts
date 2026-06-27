import { expect, it } from "vitest";

import { slugify } from "./slugify.ts";

// SPEC: story.slug.create — each test binds to a scenario via the [scenario.<id>]
// prefix on its it() title (CONVENTIONS). The engine reads the binding from THIS
// SOURCE, never from the report (which carries only each test's identity + outcome).
// Plain unit/property tests without the prefix are fine — `bindings: scoped` leaves
// them out of scope rather than failing verify.
it("[scenario.slug.create.basic] lowercases and hyphenates words", () => {
    expect(slugify("Hello World")).toBe("hello-world");
});

it("[scenario.slug.create.symbols] collapses punctuation and trims hyphens", () => {
    expect(slugify("TypeScript & You!")).toBe("typescript-you");
});
