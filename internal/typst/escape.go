package typst

import "strings"

// EscapeString escapes s for safe embedding inside a Typst string literal ("...").
// Neutralizes \, ", and # to prevent code injection.
func EscapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `#`, `\#`)
	return s
}

// EscapeContent escapes s for safe embedding inside a Typst content block ([...]).
// Neutralizes \, ], and # to prevent code injection.
func EscapeContent(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `]`, `\]`)
	s = strings.ReplaceAll(s, `#`, `\#`)
	return s
}
