package config

import (
	"encoding/json"
	"testing"
)

func TestStripJSONC(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"line comment", "{\"a\":1 // hi\n}", "{\"a\":1 \n}"},
		{"block comment", "{/* x */\"a\":1}", "{\"a\":1}"},
		{"trailing comma object", "{\"a\":1,}", "{\"a\":1}"},
		{"trailing comma array", "[1,2,]", "[1,2]"},
		{"trailing comma with space", "{\"a\":1 ,\n}", "{\"a\":1 \n}"},
		{"slashes in string kept", "{\"u\":\"http://x//y\"}", "{\"u\":\"http://x//y\"}"},
		{"comma in string kept", "{\"a\":\"x,]\"}", "{\"a\":\"x,]\"}"},
		{"escaped quote in string", "{\"a\":\"x\\\"//y\"}", "{\"a\":\"x\\\"//y\"}"},
		{"block-comment marker in string kept", "{\"a\":\"/* not a comment */\"}", "{\"a\":\"/* not a comment */\"}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := string(stripJSONC([]byte(c.in))); got != c.want {
				t.Errorf("stripJSONC(%q)\n got %q\nwant %q", c.in, got, c.want)
			}
		})
	}
}

// The stripped output must always be parseable JSON.
func TestStripJSONCParses(t *testing.T) {
	src := `{
  // a target map
  "version": 1,
  "targets": {
    "web": { "command": "pnpm test", "format": "junit", "report": "j.xml", "source": "src" }, /* trailing */
  },
}`
	var v map[string]any
	if err := json.Unmarshal(stripJSONC([]byte(src)), &v); err != nil {
		t.Fatalf("stripped JSONC did not parse: %v", err)
	}
	if v["version"].(float64) != 1 {
		t.Errorf("version = %v, want 1", v["version"])
	}
}
