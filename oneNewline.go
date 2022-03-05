package gonl

// OneNewline returns a string with exactly one terminating newline
// character. More simple than strings.TrimRight. When input string
// ends with multiple OneNewline characters, it will strip off all but
// first one, reusing the same underlying string bytes. When string
// does not end in a OneNewline character, it returns the original
// string with a OneNewline character appended. Newline characters
// before any non-OneNewline characters are ignored.
func OneNewline(s string) string {
	l := len(s)
	if l == 0 {
		return "\n"
	}

	// While this is O(length s), it stops as soon as it finds the
	// first non newline character in the string starting from the
	// right hand side of the input string. Generally this only scans
	// one or two characters and returns.

	for i := l - 1; i >= 0; i-- {
		if s[i] != '\n' {
			if i+1 < l && s[i+1] == '\n' {
				return s[:i+2]
			}
			return s[:i+1] + "\n"
		}
	}

	// The entire buffer consists of newline characters, so just
	// return the first one.
	return s[:1]
}
