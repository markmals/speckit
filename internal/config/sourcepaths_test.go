package config

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSourcePathsUnmarshalString(t *testing.T) {
	var sp SourcePaths
	if err := json.Unmarshal([]byte(`"apps/web/app"`), &sp); err != nil {
		t.Fatal(err)
	}
	if len(sp) != 1 || sp[0] != "apps/web/app" {
		t.Fatalf("string source should decode to one path, got %v", sp)
	}
}

func TestSourcePathsUnmarshalArrayTrims(t *testing.T) {
	var sp SourcePaths
	if err := json.Unmarshal([]byte(`["cmd/troved","internal"," cmd/trove-transcode "]`), &sp); err != nil {
		t.Fatal(err)
	}
	want := SourcePaths{"cmd/troved", "internal", "cmd/trove-transcode"}
	if len(sp) != len(want) || sp[0] != want[0] || sp[1] != want[1] || sp[2] != want[2] {
		t.Fatalf("array source should decode trimmed, got %v", sp)
	}
}

func TestSourcePathsUnmarshalRejectsNonString(t *testing.T) {
	var sp SourcePaths
	if err := json.Unmarshal([]byte(`123`), &sp); err == nil {
		t.Error("a number must be rejected")
	}
	if err := json.Unmarshal([]byte(`[1,2]`), &sp); err == nil {
		t.Error("an array of numbers must be rejected")
	}
}

func TestSourcePathsMarshalErgonomic(t *testing.T) {
	one, err := json.Marshal(SourcePaths{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if string(one) != `"a"` {
		t.Fatalf("one path should marshal as a bare string, got %s", one)
	}
	many, err := json.Marshal(SourcePaths{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if string(many) != `["a","b"]` {
		t.Fatalf("multiple paths should marshal as an array, got %s", many)
	}
}

func TestSourcePathsValidate(t *testing.T) {
	if errs := (SourcePaths{}).Validate("t"); len(errs) == 0 {
		t.Error("an empty source list must be invalid")
	}
	if errs := (SourcePaths{"a"}).Validate("t"); len(errs) != 0 {
		t.Errorf("a single path is valid, got %v", errs)
	}
	if errs := (SourcePaths{"a", ""}).Validate("t"); len(errs) == 0 {
		t.Error("a blank entry must be invalid")
	}
}

func TestSourcePathsRoundTrip(t *testing.T) {
	cases := []SourcePaths{{"a"}, {"a", "b", "c"}}
	for _, want := range cases {
		b, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("marshal %v: %v", want, err)
		}
		var got SourcePaths
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal %s: %v", b, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("round-trip mismatch: %v -> %s -> %v", want, b, got)
		}
	}
}
