package gonl

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

const maxInt = int(^uint(0) >> 1)
const minRead = 512
const smallBufferSize = 64

// BatchLineWriter is an io.WriteCloser that buffers output to ensure
// it only emits bytes to the underlying io.WriteCloser on line feed
// boundaries.
//
// It is important for caller to Close the BatchLineWriter to flush
// any residual data that was not terminated with a newline.
//
// Compare this structure with PerLineWriter. This structure is not
// suitable for situations that require line buffering. This structure
// is used to reduce the number of Write invocations on the underlying
// io.WriteCloser by buffering data, but calling its Write method only
// invokes Write on the underlying io.WriteCloser with a newline
// terminated sequence of bytes, potentially with more than one line
// being written at a time.
type BatchLineWriter struct {
	buf []byte // contents buf[offset:len(buf)]

	wc io.WriteCloser

	off int // read at buf[off:]; write at buf[:len(buf)]

	// Flush on LF after buffer this size or larger.
	flushThreshold int

	// -1 when no newlines in buf
	indexOfFinalNewline int
}

// NewBatchLineWriter returns a new BatchLineWriter with the specified
// flush threshold. Whenever the number of bytes in the buffer exceeds
// the specified threshold, it flushes the buffer to the underlying
// io.WriteCloser, up to and including the final LF byte.
//
//     func Example() error {
//         // Flush completed lines to os.Stdout at least every 512
//         // bytes.
//         lf, err := gonl.NewBatchLineWriter(os.Stdout, 512)
//         if err != nil {
//             return err
//         }
//
//         // Give copy buffer some room.
//         _, rerr := io.CopyBuffer(lf, os.Stdin, make([]byte, 4096))
//
//         // Clean up.
//         cerr := lf.Close()
//         if rerr == nil {
//             return cerr
//         }
//         return rerr
//     }
func NewBatchLineWriter(wc io.WriteCloser, flushThreshold int) (*BatchLineWriter, error) {
	if flushThreshold <= 0 {
		return nil, fmt.Errorf("cannot create BatchLineWriter when flushThreshold less than or equal to 0: %d", flushThreshold)
	}
	return &BatchLineWriter{
		wc:                  wc,
		flushThreshold:      flushThreshold,
		indexOfFinalNewline: -1,
	}, nil
}

func (lw *BatchLineWriter) bufferGrow(n int) int {
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
		panic(errors.New("gonl.BatchLineWriter: too large"))
	} else {
		// Allocate new backing array, then copy bytes.
		buf := make([]byte, 2*c+n)
		copy(buf, lw.buf[lw.off:])
		lw.buf = buf
	}
	lw.indexOfFinalNewline -= lw.off
	lw.off = 0
	lw.buf = lw.buf[:mpn]
	return m
}

// bufferGrowInline is an inlineable version of grow for the fast case
// where the internal buffer only needs to be resliced. It returns the
// index where bytes should be written and whether it succeeded.
func (lw *BatchLineWriter) bufferGrowInline(n int) (int, bool) {
	l := len(lw.buf)
	lpn := l + n
	if lpn <= cap(lw.buf) {
		lw.buf = lw.buf[:lpn]
		return l, true
	}
	return 0, false
}

func (lw *BatchLineWriter) bufferLength() int { return len(lw.buf) - lw.off }

func (lw *BatchLineWriter) bufferReset() {
	lw.buf = lw.buf[:0]
	lw.indexOfFinalNewline = -1
	lw.off = 0
}

// Close flushes all buffered data to the underlying io.WriteCloser,
// including bytes without a trailing LF, then closes the underlying
// io.WriteCloser. This will either return any error caused by writing
// the bytes to the underlying io.WriteCloser, or an error caused by
// closing it. Use this method when done with a BatchLineWriter to
// prevent data loss.
func (lw *BatchLineWriter) Close() error {
	var err error

	if lw.bufferLength() > 0 {
		_, err = lw.wc.Write(lw.buf[lw.off:])
		if err != nil {
			lw.bufferReset()
			_ = lw.wc.Close()
			lw.wc = nil
			return err
		}
	}

	lw.bufferReset()
	err = lw.wc.Close()
	lw.wc = nil
	return err
}

// flush flushes buffer to underlying io.WriteCloser, up to but
// excluding the specified index.
func (lw *BatchLineWriter) flush(leno, lenp, index int) (int, error) {
	debug("flush: leno: %d; len(p): %d; index: %d\n", leno, lenp, index)
	debug("flush: lw.off: %d; expected nw: %d\n", lw.off, index-lw.off)
	debug("flush: before: %q\n", lw.buf[lw.off:])
	nw, err := lw.wc.Write(lw.buf[lw.off:index])
	if nw < 0 {
		return nw, errors.New("invalid write result")
	}
	if err == nil {
		lw.off += nw                // advance offset to after nw
		lw.indexOfFinalNewline = -1 // optimization
		return lenp, nil
	}

	// nb is the number new bytes from p that got written to file.
	nb := nw - leno
	if nb >= 0 {
		// Wrote nb of the new bytes, but upstream assumes nothing
		// else was written, therefore use the opportunity to reset
		// buffer.
		lw.bufferReset()
		return nb, err
	}

	// Had nb more bytes been written, this would have broken even
	// with what was in buffer before flush was invoked. So report
	// that 0 bytes of new data was actually written, but keep the
	// bytes in the buffer that we already had.
	debug("flush: nb: %d\n", nb)
	lw.off += nw
	lw.buf = lw.buf[:lw.off-nb]
	debug("flush: after:  %q\n", lw.buf[lw.off:])

	lw.indexOfFinalNewline = bytes.LastIndexByte(lw.buf[lw.off:], '\n')
	debug("flush: indexOfFinalNewline: %d; lw.off: %d; nb: %d\n", lw.indexOfFinalNewline, lw.off, nb)
	if lw.indexOfFinalNewline != -1 {
		lw.indexOfFinalNewline += lw.off
	}

	return 0, err
}

func (lw *BatchLineWriter) Write(p []byte) (int, error) {
	leno := lw.bufferLength()

	// functionally equivalent to `lw.buf = append(lw.buf, p...)`
	m, ok := lw.bufferGrowInline(len(p))
	if !ok {
		m = lw.bufferGrow(len(p))
	}
	// Because just grew, no way this does not copy all p.
	copy(lw.buf[m:], p)

	if finalIndex := bytes.LastIndexByte(p, '\n'); finalIndex >= 0 {
		lw.indexOfFinalNewline = m + finalIndex
	}

	debug("Write: m: %d; len(p): %d; indexOfFinalNewLine: %d\n", m, len(p), lw.indexOfFinalNewline)

	// TODO Should this limit based on entire buffer size, or how much
	// data is being used by buffer. Opting for the latter here.
	if lw.bufferLength() < lw.flushThreshold || lw.indexOfFinalNewline < lw.off {
		// Either do not need to flush, or no newline exists in buffer
		debug("Write: no need to flush\n")
		return len(p), nil
	}

	// Buffer is larger than threshold, and has LF: write everything
	// up to and including that final LF.
	return lw.flush(leno, len(p), lw.indexOfFinalNewline+1)
}
