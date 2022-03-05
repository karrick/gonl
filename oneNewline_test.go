package gonl

import (
	"fmt"
	"testing"
)

func ExampleOneNewline() {
	fmt.Println(OneNewline("abc\n\ndef\n\n"))
	// Output:
	// abc
	//
	// def
}

func TestOneNewline(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if got, want := OneNewline(""), "\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
	t.Run("single character", func(t *testing.T) {
		if got, want := OneNewline("a"), "a\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
	t.Run("single newline", func(t *testing.T) {
		if got, want := OneNewline("\n"), "\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
	t.Run("multiple newline", func(t *testing.T) {
		if got, want := OneNewline("\n\n"), "\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
	t.Run("string plus single newline", func(t *testing.T) {
		if got, want := OneNewline("abc\n"), "abc\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
	t.Run("string plus multiple newlines", func(t *testing.T) {
		if got, want := OneNewline("abc\n\n"), "abc\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
	t.Run("string with multiple embedded newlines", func(t *testing.T) {
		if got, want := OneNewline("abc\n\ndef\n\n"), "abc\n\ndef\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
	t.Run("string with all newlines", func(t *testing.T) {
		if got, want := OneNewline("\n\n\n\n"), "\n"; got != want {
			t.Errorf("GOT: %q; WANT: %q", got, want)
		}
	})
}
