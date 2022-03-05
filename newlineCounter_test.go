package gonl

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func ExampleNewlineCounter() {
	c1, err := NewlineCounter(strings.NewReader("one\ntwo\nthree\n"))
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(c1)

	c2, err := NewlineCounter(strings.NewReader("one\ntwo\nthree"))
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(c2)
	// Output:
	// 3
	// 3
}

func TestNewlineCounter(t *testing.T) {
	t.Run("sans newline", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader(""))
			ensureError(t, err)
			if got, want := c, 0; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
		t.Run("one", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader("one"))
			ensureError(t, err)
			if got, want := c, 1; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
		t.Run("two", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader("one\ntwo"))
			ensureError(t, err)
			if got, want := c, 2; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
		t.Run("three", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader("one\ntwo\nthree"))
			ensureError(t, err)
			if got, want := c, 3; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
	})

	t.Run("with newline", func(t *testing.T) {
		t.Run("empty", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader("\n"))
			ensureError(t, err)
			if got, want := c, 0; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
		t.Run("one", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader("one\n"))
			ensureError(t, err)
			if got, want := c, 1; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
		t.Run("two", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader("one\ntwo\n"))
			ensureError(t, err)
			if got, want := c, 2; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
		t.Run("three", func(t *testing.T) {
			c, err := NewlineCounter(strings.NewReader("one\ntwo\nthree\n"))
			ensureError(t, err)
			if got, want := c, 3; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
		})
	})
}
