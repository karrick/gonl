package gonl

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// ErrIO represents is returned for IO errors that do not have an
// associated file system path.
type ErrIO struct {
	Op  string
	Err error
}

func (e *ErrIO) Error() string { return e.Op + ": " + e.Err.Error() }

func (e *ErrIO) Unwrap() error { return e.Err }

func ensureBuffer(tb testing.TB, got interface{ Bytes() []byte }, want string) {
	tb.Helper()
	if g, w := string(got.Bytes()), want; g != w {
		tb.Errorf("GOT: %q; WANT: %q", g, w)
	}
}

func ensureBufferLimit(tb testing.TB, buf []byte, n int, want string) {
	tb.Helper()
	if got, want := n, len(want); got != want {
		tb.Fatalf("GOT: %v; WANT: %v", got, want)
	}
	if got, want := string(buf[:n]), want; got != want {
		tb.Errorf("GOT: %v; WANT: %v", got, want)
	}
}

func ensureError(tb testing.TB, got error, contains ...string) {
	tb.Helper()
	if len(contains) == 0 || (len(contains) == 1 && contains[0] == "") {
		if got != nil {
			tb.Fatalf("GOT: %v; WANT: %v", got, contains)
		}
	} else if got == nil {
		tb.Errorf("GOT: %v; WANT: %v", got, contains)
	} else {
		for _, stub := range contains {
			m := got.Error()
			if stub != "" && !strings.Contains(m, stub) {
				tb.Errorf("GOT: %v; WANT: %q", got, stub)
			}
		}
	}
}

func ensureErrorNil(tb testing.TB, got error) {
	tb.Helper()
	if got != nil {
		tb.Fatalf("GOT: %T(%q); WANT: <nil>", got, got.Error())
	}
}

func ensureWrite(tb testing.TB, w io.Writer, p string) {
	tb.Helper()
	n, err := w.Write([]byte(p))
	if got, want := n, len(p); got != want {
		tb.Errorf("GOT: %v; WANT: %v", got, want)
	}
	ensureErrorNil(tb, err)
}

type wantState struct {
	buf                 string
	n                   int
	indexOfFinalNewline int
	isShortWrite        bool
}

func ensureWriteResponse(tb testing.TB, lfwc *BatchLineWriter, p string, state wantState) {
	tb.Helper()
	n, err := lfwc.Write([]byte(p))
	if got, want := n, state.n; got != want {
		tb.Errorf("BYTES WRITTEN: GOT: %v; WANT: %v", got, want)
	}
	if state.isShortWrite {
		ensureError(tb, err, io.ErrShortWrite.Error())
	} else {
		ensureError(tb, err)
	}
	if got, want := lfwc.buf, []byte(state.buf); !bytes.Equal(got, want) {
		tb.Errorf("GOT: %q; WANT: %q", got, want)
	}
	if got, want := lfwc.indexOfFinalNewline, state.indexOfFinalNewline; got != want {
		tb.Errorf("FINAL NEWLINE: GOT: %v; WANT: %v", got, want)
	}
}

func ensurePanic(tb testing.TB, want string, callback func()) {
	tb.Helper()
	defer func() {
		r := recover()
		if r == nil {
			tb.Fatalf("GOT: %v; WANT: %v", r, want)
			return
		}
		if got := fmt.Sprintf("%v", r); got != want {
			tb.Fatalf("GOT: %v; WANT: %v", got, want)
		}
	}()
	callback()
}
