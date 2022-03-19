package gonl

import (
	"io"
	"testing"
)

type errClose struct{}

func (ew errClose) Error() string { return "test close error" }

type errWrite struct{}

func (ew errWrite) Error() string { return "test write error" }

type errOnClose struct{}

func (eoc *errOnClose) Write(p []byte) (int, error) { return len(p), nil }
func (eoc *errOnClose) Close() error                { return errClose{} }

type errOnWrite struct{}

func (eoc *errOnWrite) Write(p []byte) (int, error) { return 0, errWrite{} }
func (eoc *errOnWrite) Close() error                { return errClose{} }

////////////////////////////////////////

func (lw *BatchLineWriter) bufferBytes() []byte { return lw.buf[lw.off:] }

func (lw *BatchLineWriter) bufferString() string { return string(lw.buf[lw.off:]) }

func (lw *BatchLineWriter) bufferWrite(p []byte) (n int, err error) {
	m, ok := lw.bufferGrowInline(len(p))
	if !ok {
		m = lw.bufferGrow(len(p))
	}
	return copy(lw.buf[m:], p), nil
}

func TestBatchLineWriter(t *testing.T) {
	// initially empty vs not empty
	// buf newline: (none, single: (at end, not at end), multiple: (at end, not at end))
	// data newline: (none, single: (at end, not at end), multiple: (at end, not at end))
	// flush vs not-flush
	// write error vs no write error

	t.Run("NewBatchLineWriter", func(t *testing.T) {
		_, err := NewBatchLineWriter(NopCloseWriter(io.Discard), 0)
		ensureError(t, err, "flushThreshold")

		_, err = NewBatchLineWriter(NopCloseWriter(io.Discard), -1)
		ensureError(t, err, "flushThreshold")
	})

	t.Run("Close", func(t *testing.T) {
		t.Run("no error", func(t *testing.T) {
			wc, err := NewBatchLineWriter(NopCloseWriter(io.Discard), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, wc, "line 1\n")
			ensureError(t, wc.Close())
		})
		t.Run("write returns error", func(t *testing.T) {
			wc, err := NewBatchLineWriter(&errOnWrite{}, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, wc, "line 1")
			ensureError(t, wc.Close(), "test write error")
		})
		t.Run("close returns error", func(t *testing.T) {
			wc, err := NewBatchLineWriter(&errOnClose{}, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, wc, "line 1")
			ensureError(t, wc.Close(), "test close error")
		})
		t.Run("write error during close", func(t *testing.T) {
			wc, err := NewBatchLineWriter(NopCloseWriter(io.Discard), 16)
			ensureErrorNil(t, err)
			ensureError(t, wc.Close())
		})
	})

	t.Run("flushCompleted", func(t *testing.T) {
		t.Run("buf has no newlines", func(t *testing.T) {
			wc, err := NewBatchLineWriter(NopCloseWriter(io.Discard), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, wc, "line 1")
			n, err := wc.flushCompleted(0, 0, 0)
			if got, want := n, 0; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
			ensureErrorNil(t, err)
		})
		t.Run("buf has newlines", func(t *testing.T) {
			output := new(testBuffer)
			wc, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, wc, "line 1\n")
			ensureWrite(t, wc, "line 2")
			const tokenInt = 42
			n, err := wc.flushCompleted(13, tokenInt, 7)
			if got, want := n, tokenInt; got != want {
				t.Errorf("GOT: %v; WANT: %v", got, want)
			}
			ensureErrorNil(t, err)
			ensureStringer(t, output, "line 1\n")
		})
	})

	t.Run("ReadFrom", func(t *testing.T) {
		r := &testReader{tuples: []tuple{
			tuple{"line 1\n", nil},
			tuple{"line 2\n", nil},
			tuple{"line 3\n", nil},
			tuple{"line 4\n", nil},
			tuple{"line 5", io.EOF},
		}}

		output := new(testBuffer)

		lw, err := NewBatchLineWriter(output, 5)
		ensureErrorNil(t, err)

		nr, err := lw.ReadFrom(r)
		ensureErrorNil(t, err)

		if got, want := nr, int64(34); got != want {
			t.Errorf("GOT: %v; WANT: %v", got, want)
		}
		ensureStringer(t, output, "line 1\nline 2\nline 3\nline 4\n")

		// flush final non terminated line
		err = lw.Close()
		ensureErrorNil(t, err)
		ensureStringer(t, output, "line 1\nline 2\nline 3\nline 4\nline 5")
	})

	t.Run("Write", func(t *testing.T) {
		t.Run("buf empty | data no newline | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)

			p := "unterminated line"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "unterminated line",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "")
		})

		t.Run("buf empty | data single newline | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)

			p := "terminated line\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 p,
				n:                   len(p),
				indexOfFinalNewline: len(p) - 1,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf empty | data single newline | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)

			p := "terminated line\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, p)
		})
		t.Run("buf empty | data single newline | at end | exact flush | no write error", func(t *testing.T) {
			// Flush when buffer is exactly full and final newline.
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)

			p := "terminated line\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, p)
		})
		t.Run("buf empty | data single newline | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)

			p := "terminated line\n"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				n:                   4,
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "term")
		})

		t.Run("buf empty | data single newline | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)

			p := "terminated\nline"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "terminated\nline",
				n:                   len(p),
				indexOfFinalNewline: 10,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf empty | data single newline | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)

			p := "terminated\nline"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "terminated\n")
		})
		t.Run("buf empty | data single newline | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)

			p := "terminated\nline"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				n:                   4,
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "term")
		})

		t.Run("buf empty | data multiple newlines | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)

			p := "terminated\nline\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 p,
				n:                   len(p),
				indexOfFinalNewline: len(p) - 1,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf empty | data multiple newlines | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)

			p := "terminated\nline\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "terminated\nline\n")
		})
		t.Run("buf empty | data multiple newlines | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)

			p := "terminated\nline\n"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				n:                   4,
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "term")
		})

		t.Run("buf empty | data multiple newlines | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)

			p := "terminated\nline\nhere"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 p,
				n:                   len(p),
				indexOfFinalNewline: 15,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf empty | data multiple newlines | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)

			p := "terminated\nline\nhere"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "here",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "terminated\nline\n")
		})
		t.Run("buf empty | data multiple newlines | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)

			p := "terminated\nline\nhere"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				n:                   4,
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "term")
		})

		//
		// buf not empty
		//

		t.Run("buf not empty | buf multiple newlines | at end | data multiple newlines | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4\n",
				n:                   len(p),
				indexOfFinalNewline: len("line 1\nline 2\n") + len(p) - 1,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data multiple newlines | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4\nline 5\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\nline 4\nline 5\n")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data multiple newlines | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)

			// Below 14 bytes does not trip threshold, so nothing
			// flushed.
			ensureWrite(t, lw, "line 1\nline 2\n")

			// The next 21 byte write will trip threshold, and only 4
			// bytes will be flushed.
			p := "line 3\nline 4\nline 5\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 " 1\nline 2\n",
				isShortWrite:        true,
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | at end | data multiple newlines | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4",
				n:                   len(p),
				indexOfFinalNewline: 20,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data multiple newlines | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4",
				n:                   len(p),
				indexOfFinalNewline: len("line 1\nline 2\nline 3\n") - 1,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data multiple newlines | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\n",
				n:                   0,
				indexOfFinalNewline: len(" 1\nline 2\n") - 1,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | at end | data no newline | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3",
				n:                   6,
				indexOfFinalNewline: 13,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data no newline | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 p,
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\n")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data no newline | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\n",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | at end | data single newline | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\n",
				n:                   7,
				indexOfFinalNewline: 20,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data single newline | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\n")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data single newline | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 " 1\nline 2\n",
				isShortWrite:        true,
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | at end | data single newline | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4",
				n:                   len(p),
				indexOfFinalNewline: 20,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data single newline | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 4",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\n")
		})
		t.Run("buf not empty | buf multiple newlines | at end | data single newline | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\n")

			p := "line 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\n",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | not at end | data multiple newlines | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 64)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4\nline 5\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4\nline 5\n",
				n:                   len(p),
				indexOfFinalNewline: 34,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data multiple newlines | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4\nline 5\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\nline 4\nline 5\n")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data multiple newlines | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4\nline 5\n"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\nline 3",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | not at end | data multiple newlines | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 64)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4\nline 5"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4\nline 5",
				n:                   len(p),
				indexOfFinalNewline: 27,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data multiple newlines | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4\nline 5"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 5",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\nline 4\n")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data multiple newlines | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4\nline 5"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\nline 3",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | not at end | data no newline | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 64)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "line 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3line 4",
				n:                   len(p),
				indexOfFinalNewline: 13,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data no newline | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "line 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 3line 4",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\n")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data no newline | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "line 4"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\nline 3",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | not at end | data single newline | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "line 4\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3line 4\n",
				n:                   len(p),
				indexOfFinalNewline: 26,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data single newline | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "line 4\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3line 4\n")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data single newline | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "line 4\n"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\nline 3",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf multiple newlines | not at end | data single newline | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\nline 3",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data single newline | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 4",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\n")
		})
		t.Run("buf not empty | buf multiple newlines | not at end | data single newline | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\nline 2\nline 3")

			p := "\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\nline 2\nline 3",
				n:                   0,
				indexOfFinalNewline: 9,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf no newline | data multiple newlines | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "\nline 2\nline 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4",
				n:                   len(p),
				indexOfFinalNewline: 20,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf no newline | data multiple newlines | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "\nline 2\nline 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 4",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\n")
		})
		t.Run("buf not empty | buf no newline | data multiple newlines | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 24)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "\nline 2\nline 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1",
				n:                   0,
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf no newline | data no newline | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "line 2"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1line 2",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "")
		})
		// t.Run("buf not empty | buf no newline | data no newline | flush | no write error", func(t *testing.T) {
		// })
		// t.Run("buf not empty | buf no newline | data no newline | flush | write error", func(t *testing.T) {
		// })

		t.Run("buf not empty | buf no newline | data single newline | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "line 2\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1line 2\n",
				n:                   len(p),
				indexOfFinalNewline: 12,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf no newline | data single newline | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "line 2\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1line 2\n")
		})
		t.Run("buf not empty | buf no newline | data single newline | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "line 2\n"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1",
				n:                   0,
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf no newline | data single newline | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "\nline 2"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2",
				n:                   len(p),
				indexOfFinalNewline: 6,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf no newline | data single newline | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "\nline 2"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 2",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\n")
		})
		t.Run("buf not empty | buf no newline | data single newline | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1")

			p := "\nline 2"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1",
				n:                   0,
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf single newline | at end | data multiple newlines | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\n",
				n:                   len(p),
				indexOfFinalNewline: 20,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf single newline | at end | data multiple newlines | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\n")
		})
		t.Run("buf not empty | buf single newline | at end | data multiple newlines | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				isShortWrite:        true,
				buf:                 " 1\n",
				n:                   0,
				indexOfFinalNewline: 2,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf single newline | at end | data multiple newlines | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3\nline 4",
				n:                   len(p),
				indexOfFinalNewline: 20,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf single newline | at end | data multiple newlines | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 4",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\nline 3\n")
		})
		t.Run("buf not empty | buf single newline | at end | data multiple newlines | not at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\nline 4"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 " 1\n",
				n:                   0,
				indexOfFinalNewline: 2,
				isShortWrite:        true,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf single newline | at end | data no newline | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2",
				n:                   len(p),
				indexOfFinalNewline: 6,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf single newline | at end | data no newline | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 2",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\n")
		})
		t.Run("buf not empty | buf single newline | at end | data no newline | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 " 1\n",
				n:                   0,
				indexOfFinalNewline: 2,
				isShortWrite:        true,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf single newline | at end | data single newline | at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\n",
				n:                   len(p),
				indexOfFinalNewline: 13,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf single newline | at end | data single newline | at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\n")
		})
		t.Run("buf not empty | buf single newline | at end | data single newline | at end | flush | write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\n"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 " 1\n",
				n:                   0,
				indexOfFinalNewline: 2,
				isShortWrite:        true,
			})
			ensureStringer(t, output, "line")
		})

		t.Run("buf not empty | buf single newline | at end | data single newline | not at end | no flush", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 32)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 1\nline 2\nline 3",
				n:                   len(p),
				indexOfFinalNewline: 13,
			})
			ensureStringer(t, output, "")
		})
		t.Run("buf not empty | buf single newline | at end | data single newline | not at end | flush | no write error", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(output, 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 "line 3",
				n:                   len(p),
				indexOfFinalNewline: -1,
			})
			ensureStringer(t, output, "line 1\nline 2\n")
		})
		t.Run("buf not empty | buf single newline | at end | data single newline | not at end | flush | write error | no new bytes", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 8)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3"
			ensureWriteResponse(t, lw, p, wantState{
				buf:                 " 1\n",
				n:                   0,
				indexOfFinalNewline: 2,
				isShortWrite:        true,
			})
			ensureStringer(t, output, "line")
		})
		t.Run("buf not empty | buf single newline | at end | data multiple newlines | at end | flush | write error | zero new bytes", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 7)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   0,
				indexOfFinalNewline: -1,
				isShortWrite:        true,
			})
			ensureStringer(t, output, "line 1\n")
		})
		t.Run("buf not empty | buf single newline | at end | data multiple newlines | at end | flush | write error | some new bytes", func(t *testing.T) {
			output := new(testBuffer)
			lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 12)), 16)
			ensureErrorNil(t, err)
			ensureWrite(t, lw, "line 1\n")

			p := "line 2\nline 3\n"
			ensureWriteResponse(t, lw, p, wantState{
				n:                   5,
				indexOfFinalNewline: -1,
				isShortWrite:        true,
			})
			ensureStringer(t, output, "line 1\nline ")
		})

		t.Run("buf not empty | buf single newline | not at end | data multiple newlines | at end", func(t *testing.T) {
			const buf = "line 1\nline 2"
			const data = "\nline 3\nline 4"

			t.Run("no flush", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 32)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 1\nline 2\nline 3\nline 4",
					n:                   len(data),
					indexOfFinalNewline: 20,
				})
				ensureStringer(t, output, "")
			})
			t.Run("write", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 16)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 4",
					n:                   len(data),
					indexOfFinalNewline: -1,
				})
				ensureStringer(t, output, "line 1\nline 2\nline 3\n")
			})
			t.Run("error", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 " 1\nline 2",
					n:                   0,
					indexOfFinalNewline: 2,
					isShortWrite:        true,
				})
				ensureStringer(t, output, "line")
			})
		})

		t.Run("buf not empty | buf single newline | not at end | data multiple newlines | not at end", func(t *testing.T) {
			const buf = "\nline 1"
			const data = "\nline 2\nline 3"

			t.Run("no flush", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 32)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "\nline 1\nline 2\nline 3",
					n:                   len(data),
					indexOfFinalNewline: 14,
				})
				ensureStringer(t, output, "")
			})
			t.Run("write", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 16)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 3",
					n:                   len(data),
					indexOfFinalNewline: -1,
				})
				ensureStringer(t, output, "\nline 1\nline 2\n")
			})
			t.Run("error", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "e 1",
					n:                   0,
					indexOfFinalNewline: -1,
					isShortWrite:        true,
				})
				ensureStringer(t, output, "\nlin")
			})
		})

		t.Run("buf not empty | buf single newline | not at end | data no newline", func(t *testing.T) {
			const buf = "line 1\nline 2"
			const data = "line 3"

			t.Run("no flush", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 32)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 1\nline 2line 3",
					n:                   len(data),
					indexOfFinalNewline: 6,
				})
				ensureStringer(t, output, "")
			})
			t.Run("write", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 8)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 2line 3",
					n:                   len(data),
					indexOfFinalNewline: -1,
				})
				ensureStringer(t, output, "line 1\n")
			})
			t.Run("error", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 " 1\nline 2",
					n:                   0,
					indexOfFinalNewline: 2,
					isShortWrite:        true,
				})
				ensureStringer(t, output, "line")
			})
		})

		t.Run("buf not empty | buf single newline | not at end | data single newline | at end", func(t *testing.T) {
			const buf = "line 1\nline 2"
			const data = "line 3\n"

			t.Run("no flush", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 32)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 1\nline 2line 3\n",
					n:                   len(data),
					indexOfFinalNewline: 19,
				})
				ensureStringer(t, output, "")
			})
			t.Run("write", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 8)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					n:                   len(data),
					indexOfFinalNewline: -1,
				})
				ensureStringer(t, output, "line 1\nline 2line 3\n")
			})
			t.Run("error", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 " 1\nline 2",
					n:                   0,
					indexOfFinalNewline: 2,
					isShortWrite:        true,
				})
				ensureStringer(t, output, "line")
			})
		})

		t.Run("buf not empty | buf single newline | not at end | data single newline | not at end", func(t *testing.T) {
			const buf = "line 1\nline 2"
			const data = "\nline 3"

			t.Run("no flush", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 32)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 1\nline 2\nline 3",
					n:                   len(data),
					indexOfFinalNewline: 13,
				})
				ensureStringer(t, output, "")
			})
			t.Run("write", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(output, 8)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 "line 3",
					n:                   len(data),
					indexOfFinalNewline: -1,
				})
				ensureStringer(t, output, "line 1\nline 2\n")
			})
			t.Run("error", func(t *testing.T) {
				output := new(testBuffer)
				lw, err := NewBatchLineWriter(NopCloseWriter(ShortWriter(output, 4)), 16)
				ensureErrorNil(t, err)
				ensureWrite(t, lw, buf)

				ensureWriteResponse(t, lw, data, wantState{
					buf:                 " 1\nline 2",
					n:                   0,
					indexOfFinalNewline: 2,
					isShortWrite:        true,
				})
				ensureStringer(t, output, "line")
			})
		})
	})
}

// flushCompleted writes all completed lines in buffer to underlying
// io.WriteCloser. The final incomplete line will remain in the
// buffer.
func (lw *BatchLineWriter) flushCompleted(olen, dlen, index int) (int, error) {
	if lw.indexOfFinalNewline == -1 {
		return 0, nil // buffer has no completed lines
	}
	return lw.flush(olen, dlen, lw.indexOfFinalNewline+1)
}
