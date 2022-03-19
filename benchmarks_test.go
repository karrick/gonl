package gonl

import (
	_ "embed"
	"errors"
	"io"
	"strings"
	"testing"
)

//go:embed 2600-h.htm
var novel string

// copyBuffer is a modified version of similarly named function in
// standard library, provided here so we can prevent it from using
// ReaderFrom.
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	var written int64
	var err error

	if buf == nil {
		buf = make([]byte, 32*1024) // use same size as copy
	}

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

func BenchmarkBatchLineWriter(b *testing.B) {
	const threshold = 1024

	b.Run("ReaderFrom", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			output, err := NewBatchLineWriter(new(testBuffer), threshold)
			if err != nil {
				b.Fatal(err)
			}

			_, err = output.ReadFrom(strings.NewReader(novel))
			if err != nil {
				b.Fatal(err)
			}

			if err = output.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Write", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			output, err := NewBatchLineWriter(new(testBuffer), threshold)
			if err != nil {
				b.Fatal(err)
			}

			_, err = copyBuffer(output, strings.NewReader(novel), nil)
			if err != nil {
				b.Fatal(err)
			}

			if err = output.Close(); err != nil {
				b.Fatal(err)
			}
		}
	})
}
