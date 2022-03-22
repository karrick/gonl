package gonl

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// bufSize will be used when making buffers for copying byte slices,
// and its size is chosen based on what io.Copy will create when given
// an empty slice.
const bufSize = 32 * 1024

// copyBuffer is a modified version of similarly named function in
// standard library, provided here to be able to test using
// io.CopyBuffer both with and without sending it a destination
// structure that implements io.ReaderFrom.
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	var written int64
	var err error

	if buf == nil {
		buf = make([]byte, bufSize)
	}

	// See copyBuffer in io standard library.
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("errInvalidWrite")
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func BenchmarkCheapWrites(b *testing.B) {
	b.Run("BatchLineLineWriter", func(b *testing.B) {
		// These functions contrasts the benefit of BatchLineWriter
		// having ReadFrom method available rather than only having
		// Write method.

		b.Run("ReadFrom", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				drain := new(discardWriteCloser)

				output, err := NewBatchLineWriter(drain, bufSize)
				if err != nil {
					b.Fatal(err)
				}

				_, err = output.ReadFrom(bytes.NewReader(novel))
				if err != nil {
					b.Fatal(err)
				}

				if err = output.Close(); err != nil {
					b.Fatal(err)
				}

				if got, want := drain.count, len(novel); got != want {
					b.Errorf("GOT: %v; WANT: %v", got, want)
				}
			}
		})
		b.Run("Write", func(b *testing.B) {
			buf := make([]byte, bufSize)

			for i := 0; i < b.N; i++ {
				drain := new(discardWriteCloser)

				output, err := NewBatchLineWriter(drain, bufSize)
				if err != nil {
					b.Fatal(err)
				}

				_, err = copyBuffer(output, bytes.NewReader(novel), buf)
				if err != nil {
					b.Fatal(err)
				}

				if err = output.Close(); err != nil {
					b.Fatal(err)
				}

				if got, want := drain.count, len(novel); got != want {
					b.Errorf("GOT: %v; WANT: %v", got, want)
				}
			}
		})
	})

	b.Run("PerLineWriter", func(b *testing.B) {
		b.Run("Write", func(b *testing.B) {
			buf := make([]byte, bufSize)

			for i := 0; i < b.N; i++ {
				drain := new(discardWriteCloser)
				output := &PerLineWriter{WC: drain}

				_, err := copyBuffer(output, bytes.NewReader(novel), buf)
				if err != nil {
					b.Fatal(err)
				}

				if err = output.Close(); err != nil {
					b.Fatal(err)
				}

				if got, want := drain.count, len(novel); got != want {
					b.Errorf("GOT: %v; WANT: %v", got, want)
				}
			}
		})
	})
}

func BenchmarkWorkingWrites(b *testing.B) {
	// ??? not really worried about true message authentication
	// codes. Just want to shove data into an io.Writer that does a
	// bit of work, while also verifying every byte passed through the
	// intermediate structures.
	var key = []byte("this is a dummy key")
	var mac = []byte("\xfav\x96\xd1C\xea\xb4\xdd߿\xd0G\x0e\x95\xa8)\xb5\xed\xe6\x11{e\xf2f\xd2\xea\xf5\xdb=\xb46\xff")

	b.Run("BatchLineLineWriter", func(b *testing.B) {
		b.Run("ReadFrom", func(b *testing.B) {
			drain := newHashWriteCloser(key)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				output, err := NewBatchLineWriter(drain, bufSize)
				if err != nil {
					b.Fatal(err)
				}

				_, err = output.ReadFrom(bytes.NewReader(novel))
				if err != nil {
					b.Fatal(err)
				}

				if err = output.Close(); err != nil {
					b.Fatal(err)
				}

				if !drain.ValidMAC(mac) {
					b.Errorf("Invalid MAC: %q", drain.MAC())
				}

				drain.Reset()
			}
		})
		b.Run("Write", func(b *testing.B) {
			buf := make([]byte, bufSize)
			drain := newHashWriteCloser(key)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				output, err := NewBatchLineWriter(drain, bufSize)
				if err != nil {
					b.Fatal(err)
				}

				_, err = copyBuffer(output, bytes.NewReader(novel), buf)
				if err != nil {
					b.Fatal(err)
				}

				if err = output.Close(); err != nil {
					b.Fatal(err)
				}

				if !drain.ValidMAC(mac) {
					b.Errorf("Invalid MAC: %q", drain.MAC())
				}

				drain.Reset()
			}
		})
	})

	b.Run("PerLineWriter", func(b *testing.B) {
		b.Run("Write", func(b *testing.B) {
			buf := make([]byte, bufSize)
			drain := newHashWriteCloser(key)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				output := &PerLineWriter{WC: drain}

				_, err := copyBuffer(output, bytes.NewReader(novel), buf)
				if err != nil {
					b.Fatal(err)
				}

				if err = output.Close(); err != nil {
					b.Fatal(err)
				}

				if !drain.ValidMAC(mac) {
					b.Errorf("Invalid MAC: %q", drain.MAC())
				}

				drain.Reset()
			}
		})
	})
}
