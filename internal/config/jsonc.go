package config

// stripJSONC converts JSONC (JSON with // and /* */ comments and trailing
// commas) to plain JSON. String literals are preserved verbatim, so a // or a
// trailing-looking comma inside a string is untouched. No external dependency —
// it's a two-pass, string-aware scan.
func stripJSONC(b []byte) []byte {
	return stripTrailingCommas(stripComments(b))
}

func stripComments(b []byte) []byte {
	out := make([]byte, 0, len(b))
	inStr := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			out = append(out, c)
			if c == '\\' && i+1 < len(b) { // keep the escaped char verbatim
				out = append(out, b[i+1])
				i++
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		switch {
		case c == '"':
			inStr = true
			out = append(out, c)
		case c == '/' && i+1 < len(b) && b[i+1] == '/': // line comment
			for i < len(b) && b[i] != '\n' {
				i++
			}
			if i < len(b) {
				out = append(out, '\n') // keep the newline so line numbers hold
			}
		case c == '/' && i+1 < len(b) && b[i+1] == '*': // block comment
			i += 2
			for i < len(b) && !(b[i] == '*' && i+1 < len(b) && b[i+1] == '/') {
				i++
			}
			i++ // skip '*'; the loop's i++ skips the closing '/'
		default:
			out = append(out, c)
		}
	}
	return out
}

func stripTrailingCommas(b []byte) []byte {
	out := make([]byte, 0, len(b))
	inStr := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			out = append(out, c)
			if c == '\\' && i+1 < len(b) {
				out = append(out, b[i+1])
				i++
			} else if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			out = append(out, c)
			continue
		}
		if c == ',' {
			j := i + 1
			for j < len(b) && isSpace(b[j]) {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue // drop the trailing comma
			}
		}
		out = append(out, c)
	}
	return out
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
