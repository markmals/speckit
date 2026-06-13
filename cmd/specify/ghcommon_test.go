package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirmAction(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		assumeYes bool
		want      bool
		wantErr   bool
	}{
		{"yes", "y\n", false, true, false},
		{"yes-word", "yes\n", false, true, false},
		{"no", "n\n", false, false, false},
		{"empty-defaults-no", "\n", false, false, false},
		{"garbage-is-no", "maybe\n", false, false, false},
		{"assume-yes-skips-prompt", "", true, true, false},
		{"no-input-fails-safe", "", false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := confirmAction(strings.NewReader(tc.input), &out, "Proceed?", tc.assumeYes)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("confirmAction = %v, want %v", got, tc.want)
			}
			// assume-yes must not emit a prompt.
			if tc.assumeYes && out.Len() != 0 {
				t.Errorf("assumeYes still prompted: %q", out.String())
			}
		})
	}
}
