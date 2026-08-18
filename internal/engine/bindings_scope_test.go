package engine

import "testing"

// Each binding form is read only from its language's files: a Vitest title or a
// Swift trait quoted inside a .go string literal (a Go test embedding a snippet)
// must not bind, while the same text in its own language's file does.
//
// SPEC: story.engine.verify (scenario.engine.verify.language-scoped-binding-forms)
// [scenario.engine.verify.language-scoped-binding-forms]
func TestBindingFormsAreLanguageScoped(t *testing.T) {
	vitest := `it("[scenario.x.y] proves the roundtrip", () => {})`
	swift := "@Test(.scenario(\"scenario.a.b\")) func `proves`() {}"

	goSrc := "package x\n\nvar snippet = `\n" + vitest + "\n" + swift + "\n`\n"
	if bs := bindingsInContent("pkg/x_test.go", goSrc); len(bs) != 0 {
		t.Errorf("Vitest/Swift forms inside a .go file must not bind, got %+v", bs)
	}

	if bs := bindingsInContent("app/x.test.ts", vitest+"\n"); len(bs) != 1 || bs[0].Scenario != "scenario.x.y" {
		t.Errorf(".ts Vitest title must bind scenario.x.y, got %+v", bs)
	}

	if bs := bindingsInContent("Tests/T.swift", swift+"\n"); len(bs) != 1 || bs[0].Scenario != "scenario.a.b" {
		t.Errorf(".swift trait must bind scenario.a.b, got %+v", bs)
	}
}
