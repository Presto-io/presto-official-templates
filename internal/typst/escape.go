package typst

import "strings"

var (
	stringEscaper  = strings.NewReplacer(`\`, `\\`, `"`, `\"`, `#`, `\#`)
	contentEscaper = strings.NewReplacer(`\`, `\\`, `]`, `\]`, `#`, `\#`)
)

// EscapeString escapes s for safe embedding inside a Typst string literal ("...").
// Neutralizes \, ", and # to prevent code injection.
func EscapeString(s string) string {
	return stringEscaper.Replace(s)
}

// EscapeContent escapes s for safe embedding inside a Typst content block ([...]).
// Neutralizes \, ], and # to prevent code injection.
func EscapeContent(s string) string {
	return contentEscaper.Replace(s)
}
