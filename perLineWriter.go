package gonl

import (
	"bytes"
	"errors"
	"io"
)

// PerLineWriter is a synchronous io.WriteCloser which writes each
// completed newline terminated line to the underlying io.WriteCloser.
//
// This stream processor ensures there is exactly one Write call made
// to the underlying io.WriteCloser for each newline terminated line
// being written to it.
//
// Compare this structure with BatchLineWriter. This structure is
// suitable for situations that require line buffering. This structure
// is used to ensure each newline terminated line is individually sent
// to the underlying io.WriteCloser. Calling its Write method only
// invokes Write on the underlying io.WriteCloser with a newline
// terminated sequence of bytes.
type PerLineWriter struct {
	buf []byte

	// WC is io.WriteCloser where data is ultimately written.
	WC io.WriteCloser

	off int // read at buf[off:]; write at buf[:len(buf)]
}

// NewPerLineWriter returns a new PerLineWriter that individually
// writes each newline terminated line to the provided
// io.WriteCloser. When a PerLineWriter is closed, it flushes any
// remaining bytes, then closes the provided io.WriteCloser.
func NewPerLineWriter(wc io.WriteCloser) *PerLineWriter {
	return &PerLineWriter{WC: wc}
}

// bufferGrow will ensure the backing buffer has enough room to hold
// at least n more bytes, reslicing the data in the buffer if
// possible, and expanding the backing array if necessary. It returns
// the index into the buffer where bytes may be added.
func (lw *PerLineWriter) bufferGrow(n int) int {
	m := lw.bufferLength()
	if m == 0 && lw.off != 0 {
		// Reset buffer to reduce likelihood of unnecessary
		// allocation.
		lw.bufferReset()
	}
	if i, ok := lw.bufferGrowInline(n); ok {
		// NOTE: This is the only way to exit this method with lw.off
		// potentially not being set to 0.
		return i
	}
	// NOTE: If we get here, there is no way of leaving this method
	// without lw.off set to 0, and any used portion of buffer moved
	// to the left.
	if lw.buf == nil && n <= smallBufferSize {
		lw.buf = make([]byte, n, smallBufferSize)
		return 0
	}
	mpn := m + n
	c := cap(lw.buf)
	if mpn <= c/2 {
		// If amount of room needed is less than half slice capacity,
		// slide the data over to avoid too frequent allocation and
		// byte copying.
		copy(lw.buf, lw.buf[lw.off:])
	} else if c > maxInt-c-n {
		panic(errors.New("gonl.PerLineWriter: too large"))
	} else {
		// Allocate new backing array, then copy bytes.
		buf := make([]byte, 2*c+n)
		copy(buf, lw.buf[lw.off:])
		lw.buf = buf
	}
	lw.off = 0
	lw.buf = lw.buf[:mpn]
	return m
}

// bufferGrowInline is an inlineable version of grow for the fast case
// where the internal buffer only needs to be resliced. It returns the
// index where bytes should be written and whether it succeeded.
func (lw *PerLineWriter) bufferGrowInline(n int) (int, bool) {
	l := len(lw.buf)
	lpn := l + n
	if lpn <= cap(lw.buf) {
		lw.buf = lw.buf[:lpn]
		return l, true
	}
	return 0, false
}

// bufferLength returns the number of bytes the buffer holds, ignoring
// the data already processed. Similar to len(lw.buf) for a
// non-sliding buffer.
func (lw *PerLineWriter) bufferLength() int { return len(lw.buf) - lw.off }

// bufferReset resets the buffer so it does not have any usable bytes,
// but keeps the allocated backing array.
func (lw *PerLineWriter) bufferReset() {
	lw.buf = lw.buf[:0]
	lw.off = 0
}

// Close will transform then write any data remaining in the
// PerLineWriter that was not newline terminated, then closes the
// underlying io.WriteCloser.
func (lw *PerLineWriter) Close() error {
	var err error

	if lw.bufferLength() > 0 {
		// When additional bytes are available to be written, flush
		// them without a newline before we close the stream.
		if _, err = lw.WC.Write(lw.buf[lw.off:]); err != nil {
			_ = lw.WC.Close()
			lw.WC = nil
			lw.buf = nil
			lw.off = 0
			return err
		}
	}

	err = lw.WC.Close()
	lw.WC = nil
	lw.buf = nil
	lw.off = 0
	return err
}

// Write invokes Write on the underlying io.WriteCloser for each
// newline terminated sequence of bytes in p. Each call to this method
// may result in 0, 1, or many Write calls to the underlying
// io.WriteCloser, depending on how many newline characters are in p.
func (lw *PerLineWriter) Write(p []byte) (int, error) {
	var err error
	var index int

	m, ok := lw.bufferGrowInline(len(p))
	if !ok {
		m = lw.bufferGrow(len(p))
	}
	copy(lw.buf[m:], p)
	// POST: lw.buf[m:] is new data, however lw.buf[lw.off:m] also
	// needs processing.

	// We know remaining bytes lw.buf[lw.off:m] does not have a
	// newline, so start searching at offset m.
	index = bytes.IndexByte(lw.buf[m:], '\n')
	if index == -1 {
		return len(p), nil
	}
	// POST: lw.buf[m+index] is a newline.
	index += m + 1 // extra byte to include newline

	for {
		if _, err = lw.WC.Write(lw.buf[lw.off:index]); err != nil {
			return len(p), err // ???
		}
		lw.off = index // advance buf to consume bytes processed
		index = bytes.IndexByte(lw.buf[lw.off:], '\n')
		if index == -1 {
			return len(p), nil
		}
		index += lw.off + 1 // extra byte to include newline
	}
}
