package gonl

import (
	"testing"
)

func TestPerLineWriter(t *testing.T) {
	t.Run("buffer size 0", func(t *testing.T) {
		bb := new(testBuffer)

		lw := NewPerLineWriter(bb, 0)

		nw, err := lw.Write([]byte("line1"))
		ensureSame(t, nw, 5)
		ensureErrorNil(t, err)

		// nothing written because no newline yet
		ensureSame(t, bb.String(), "")

		nw, err = lw.Write([]byte("\nline2"))
		ensureSame(t, nw, 6)
		ensureErrorNil(t, err)

		ensureSame(t, bb.String(), "line1\n")

		err = lw.Close()
		ensureErrorNil(t, err)

		ensureSame(t, bb.String(), "line1\nline2")
	})

	t.Run("buffer size 3", func(t *testing.T) {
		bb := new(testBuffer)
		const bufsize = 3 // only represents initial size; does not limit

		lw := NewPerLineWriter(bb, bufsize)

		nw, err := lw.Write([]byte("line1"))
		ensureSame(t, nw, 5)
		ensureErrorNil(t, err)

		// nothing written because no newline yet
		ensureSame(t, bb.String(), "")

		nw, err = lw.Write([]byte("\nline2"))
		ensureSame(t, nw, 6)
		ensureErrorNil(t, err)

		ensureSame(t, bb.String(), "line1\n")

		err = lw.Close()
		ensureErrorNil(t, err)

		ensureSame(t, bb.String(), "line1\nline2")
	})
}
