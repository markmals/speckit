package reports

import (
	"bufio"
	"bytes"
	"encoding/json"
)

// ParseSwiftEvents parses Swift Testing's `--event-stream-output-path` NDJSON.
// Per the spike, xunit drops the scenario binding, so the event stream is the
// apple outcome source: `test` records carry the displayName (the raw-identifier
// description, the join identity), and an `issueRecorded` event marks a failure.
func ParseSwiftEvents(data []byte) ([]Result, error) {
	idToName := map[string]string{}
	failed := map[string]bool{}

	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		var r struct {
			Kind    string `json:"kind"`
			Payload struct {
				Kind        string `json:"kind"`
				ID          string `json:"id"`
				TestID      string `json:"testID"`
				DisplayName string `json:"displayName"`
			} `json:"payload"`
		}
		if json.Unmarshal(sc.Bytes(), &r) != nil {
			continue
		}
		switch r.Kind {
		case "test":
			if r.Payload.DisplayName != "" {
				idToName[r.Payload.ID] = r.Payload.DisplayName
			}
		case "event":
			if r.Payload.Kind == "issueRecorded" {
				failed[r.Payload.TestID] = true
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(idToName))
	for id, name := range idToName {
		results = append(results, Result{Name: name, Pass: !failed[id]})
	}
	return results, nil
}
