package gonl

import (
	"bytes"
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

	t.Run("digest", func(t *testing.T) {
		// ??? not really worried about true message authentication
		// codes. Just want to shove data into an io.Writer that does a
		// bit of work, while also verifying every byte passed through the
		// intermediate structures.
		var key = []byte("this is a dummy key")
		var mac = []byte("\xfav\x96\xd1C\xea\xb4\xdd߿\xd0G\x0e\x95\xa8)\xb5\xed\xe6\x11{e\xf2f\xd2\xea\xf5\xdb=\xb46\xff")

		t.Run("ReadFrom", func(t *testing.T) {
			drain := newHashWriteCloser(key)

			output := &PerLineWriter{WC: drain}

			_, err := output.ReadFrom(bytes.NewReader(novel))
			if err != nil {
				t.Fatal(err)
			}

			if err = output.Close(); err != nil {
				t.Fatal(err)
			}

			if !drain.ValidMAC(mac) {
				t.Errorf("Invalid MAC: %q", drain.MAC())
			}

			drain.Reset()
		})

		t.Run("Write", func(t *testing.T) {
			buf := make([]byte, bufSize)
			drain := newHashWriteCloser(key)

			output := &PerLineWriter{WC: drain}

			_, err := copyBuffer(output, bytes.NewReader(novel), buf)
			if err != nil {
				t.Fatal(err)
			}

			if err = output.Close(); err != nil {
				t.Fatal(err)
			}

			if !drain.ValidMAC(mac) {
				t.Errorf("Invalid MAC: %q", drain.MAC())
			}

			drain.Reset()
		})
	})
}
