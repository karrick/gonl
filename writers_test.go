package gonl

// These structures copied from https://github.com/karrick/gorill
// project.

import "io"

// dumpWriteCloser simply tracks how many bytes have been written to
// it.
type dumpWriteCloser struct {
	count int
}

func (dw *dumpWriteCloser) Close() error {
	return nil
}

func (dw *dumpWriteCloser) Write(p []byte) (int, error) {
	dw.count += len(p)
	return len(p), nil
}

// NopCloseWriter returns a structure that implements io.WriteCloser,
// but provides a no-op Close method.  It is useful when you have an
// io.Writer that needs to be passed to a method that requires an
// io.WriteCloser.  It is the counter-part to ioutil.NopCloser, but
// for io.Writer.
//
//   wc := gonl.NopCloseWriter(w)
//   _ = wc.Close() // does nothing; always returns nil
func NopCloseWriter(wc io.Writer) io.WriteCloser {
	return nopCloseWriter{wc}
}

func (nopCloseWriter) Close() error { return nil }

type nopCloseWriter struct{ io.Writer }

// ShortWriter returns a structure that wraps an io.Writer, but
// returns io.ErrShortWrite when the number of bytes to write exceeds
// a preset limit.
func ShortWriter(w io.Writer, max int) io.Writer {
	return shortWriter{Writer: w, max: max}
}

func (s shortWriter) Write(data []byte) (int, error) {
	var short bool
	index := len(data)
	if index > s.max {
		index = s.max
		short = true
	}
	n, err := s.Writer.Write(data[:index])
	if short {
		return n, io.ErrShortWrite
	}
	return n, err
}

type shortWriter struct {
	io.Writer
	max int
}
