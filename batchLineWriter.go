package gonl

import (
	"bytes"
	"fmt"
	"io"
)

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
	buf []byte

	wc io.WriteCloser

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

// Close flushes all buffered data to the underlying io.WriteCloser,
// including bytes without a trailing LF, then closes the underlying
// io.WriteCloser. This will either return any error caused by writing
// the bytes to the underlying io.WriteCloser, or an error caused by
// closing it. Use this method when done with a BatchLineWriter to
// prevent data loss.
func (lw *BatchLineWriter) Close() error {
	_, werr := lw.wc.Write(lw.buf)
	lw.buf = nil
	lw.indexOfFinalNewline = -1
	cerr := lw.wc.Close()
	lw.wc = nil
	if werr != nil {
		return werr
	}
	return cerr
}

// flush flushes buffer to underlying io.WriteCloser, up to and
// including specified index.
func (lw *BatchLineWriter) flush(olen, dlen, index int) (int, error) {
	nw, err := lw.wc.Write(lw.buf[:index])
	if nw > 0 {
		nc := copy(lw.buf, lw.buf[nw:])
		lw.buf = lw.buf[:nc]
	}
	if err == nil {
		lw.indexOfFinalNewline -= nw
		return dlen, nil
	}
	// nb is the number new bytes from p that got written to file.
	nb := nw - olen
	if nb < 0 {
		lw.buf = lw.buf[:-nb]
		nb = 0
	} else {
		lw.buf = lw.buf[:0]
	}
	lw.indexOfFinalNewline = bytes.LastIndexByte(lw.buf, '\n')
	return nb, err
}

// Write appends bytes from p to the internal buffer, flushing buffer
// up to and including the final LF when buffer length exceeds
// threshold specified when creating the BatchLineWriter.
func (lw *BatchLineWriter) Write(p []byte) (int, error) {
	olen := len(lw.buf)
	lw.buf = append(lw.buf, p...)

	if finalIndex := bytes.LastIndexByte(p, '\n'); finalIndex >= 0 {
		lw.indexOfFinalNewline = olen + finalIndex
	}

	if len(lw.buf) < lw.flushThreshold || lw.indexOfFinalNewline < 0 {
		// Either do not need to flush, or no newline in buffer
		return len(p), nil
	}

	// Buffer larger than threshold, and has LF: write everything up
	// to and including that final LF.
	return lw.flush(olen, len(p), lw.indexOfFinalNewline+1)
}
