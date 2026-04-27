package loghelper

import "testing"

func TestSafe_PassesPlainAscii(t *testing.T) {
	in := "no special chars"
	if got := Safe(in); got != in {
		t.Errorf("Safe(%q) = %q, want unchanged", in, got)
	}
}

func TestSafe_EscapesNewline(t *testing.T) {
	in := "line1\nline2"
	want := "line1\\nline2"
	if got := Safe(in); got != want {
		t.Errorf("Safe(%q) = %q, want %q", in, got, want)
	}
}

func TestSafe_EscapesCarriageReturn(t *testing.T) {
	in := "a\rb"
	want := "a\\rb"
	if got := Safe(in); got != want {
		t.Errorf("Safe(%q) = %q, want %q", in, got, want)
	}
}

func TestSafe_EscapesTab(t *testing.T) {
	in := "a\tb"
	want := "a\\tb"
	if got := Safe(in); got != want {
		t.Errorf("Safe(%q) = %q, want %q", in, got, want)
	}
}

func TestSafe_EscapesAllControlCharsTogether(t *testing.T) {
	in := "x\ny\rz\t!"
	want := "x\\ny\\rz\\t!"
	if got := Safe(in); got != want {
		t.Errorf("Safe(%q) = %q, want %q", in, got, want)
	}
}

func TestSafe_EmptyString(t *testing.T) {
	if got := Safe(""); got != "" {
		t.Errorf("Safe(\"\") = %q, want empty", got)
	}
}
