package reports

import "encoding/xml"

// ParseJUnit parses junit-family XML (Vitest for web; Gradle for android) into
// normalized results. A <testcase> with a <failure> or <error> child failed; a
// <skipped> child is treated as not-passing.
func ParseJUnit(data []byte) ([]Result, error) {
	var doc struct {
		Cases []struct {
			Classname string    `xml:"classname,attr"`
			Name      string    `xml:"name,attr"`
			Failure   *struct{} `xml:"failure"`
			Error     *struct{} `xml:"error"`
			Skipped   *struct{} `xml:"skipped"`
		} `xml:"testsuite>testcase"`
	}
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	results := make([]Result, 0, len(doc.Cases))
	for _, c := range doc.Cases {
		results = append(results, Result{
			Suite: c.Classname,
			Name:  c.Name,
			Pass:  c.Failure == nil && c.Error == nil && c.Skipped == nil,
		})
	}
	return results, nil
}
