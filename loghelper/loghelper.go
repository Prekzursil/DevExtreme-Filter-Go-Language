// Package loghelper provides safe sanitization for user-controlled values
// before they are emitted to logs.
//
// CodeQL flags “log.Printf“ calls that interpolate user input directly via
// “%s“ because a malicious actor could inject CR/LF or control characters
// to forge log entries (CWE-117). The “Safe“ function strips/escapes
// these characters without changing the legitimate human-readable shape
// of the value.
package loghelper

import "strings"

// Safe returns “s“ with newline, carriage return, and tab characters
// replaced by their visible “\n“, “\r“, “\t“ escape sequences. This
// breaks log-injection attacks while keeping the value readable in logs.
func Safe(s string) string {
	return strings.NewReplacer(
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	).Replace(s)
}
