// Package testutil provides shared test helpers.
package testutil

import "testing"

// EqErr checks whether err matches the expected message msg.
// Returns true only when both are empty (no error expected, none received).
func EqErr(t *testing.T, id string, err error, msg string) bool {
	t.Helper()
	if err == nil && msg == "" {
		return true
	}
	if err == nil && msg != "" {
		t.Errorf("case [%s]: expected error message %q; but got nothing", id, msg)
		return false
	}
	if err != nil && msg != err.Error() {
		t.Errorf("case [%s]: expected error message %q; but got %q", id, msg, err)
	}
	return false
}

// Ptr returns a pointer to the given string.
func Ptr(s string) *string { return &s }

// Str returns the string value of a *string, or "<nil>" if nil.
func Str(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// EqPtr returns whether two *string values are equal.
func EqPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
