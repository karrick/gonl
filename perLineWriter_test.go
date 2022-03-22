package gonl

import (
	"testing"
)

func TestPerLineWriter(t *testing.T) {
	t.Run("buffer size 0", func(t *testing.T) {
		bb := new(testBuffer)

		lw := PerLineWriter{WC: bb}

		nw, err := lw.Write([]byte("line1"))
		if got, want := nw, 5; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
		ensureErrorNil(t, err)

		// nothing written because no newline yet
		if got, want := bb.String(), ""; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		nw, err = lw.Write([]byte("\nline2"))
		if got, want := nw, 6; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
		ensureErrorNil(t, err)

		if got, want := bb.String(), "line1\n"; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		err = lw.Close()
		ensureErrorNil(t, err)

		if got, want := bb.String(), "line1\nline2"; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
	})

	t.Run("buffer size 3", func(t *testing.T) {
		bb := new(testBuffer)
		const bufsize = 3 // only represents initial size; does not limit

		lw := PerLineWriter{WC: bb}

		nw, err := lw.Write([]byte("line1"))
		if got, want := nw, 5; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
		ensureErrorNil(t, err)

		// nothing written because no newline yet
		if got, want := bb.String(), ""; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		nw, err = lw.Write([]byte("\nline2"))
		if got, want := nw, 6; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
		ensureErrorNil(t, err)

		if got, want := bb.String(), "line1\n"; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}

		err = lw.Close()
		ensureErrorNil(t, err)

		if got, want := bb.String(), "line1\nline2"; got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
	})
}
