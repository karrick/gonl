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
	// Accumulated bytes waiting for newline.
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
func NewBatchLineWriter(iowc io.WriteCloser, flushThreshold int) (*BatchLineWriter, error) {
	if flushThreshold <= 0 {
		return nil, fmt.Errorf("cannot create BatchLineWriter when flushThreshold less than or equal to 0: %d", flushThreshold)
	}
	return &BatchLineWriter{
		wc:                  iowc,
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
func (lbf *BatchLineWriter) Close() error {
	_, we := lbf.wc.Write(lbf.buf)
	lbf.buf = nil
	lbf.indexOfFinalNewline = -1
	ce := lbf.wc.Close()
	lbf.wc = nil
	if we != nil {
		return we
	}
	if ce != nil {
		return ce
	}
	return nil
}

// flush flushes buffer to underlying io.WriteCloser, up to and
// including specified index.
func (lbf *BatchLineWriter) flush(olen, dlen, index int) (int, error) {
	nw, err := lbf.wc.Write(lbf.buf[:index])
	if nw > 0 {
		nc := copy(lbf.buf, lbf.buf[nw:])
		lbf.buf = lbf.buf[:nc]
	}
	if err == nil {
		lbf.indexOfFinalNewline -= nw
		return dlen, nil
	}
	// nb is the number new bytes from p that got written to file.
	nb := nw - olen
	if nb < 0 {
		lbf.buf = lbf.buf[:-nb]
		nb = 0
	} else {
		lbf.buf = lbf.buf[:0]
	}
	lbf.indexOfFinalNewline = bytes.LastIndexByte(lbf.buf, '\n')
	return nb, err
}

// Write appends bytes from p to the internal buffer, flushing buffer
// up to and including the final LF when buffer length exceeds
// threshold specified when creating the BatchLineWriter.
func (lbf *BatchLineWriter) Write(p []byte) (int, error) {
	olen := len(lbf.buf)
	lbf.buf = append(lbf.buf, p...)

	if finalIndex := bytes.LastIndexByte(p, '\n'); finalIndex >= 0 {
		lbf.indexOfFinalNewline = olen + finalIndex
	}

	if len(lbf.buf) <= lbf.flushThreshold || lbf.indexOfFinalNewline < 0 {
		// Either do not need to flush, or no newline in buffer
		return len(p), nil
	}

	// Buffer larger than threshold, and has LF: write everything up
	// to and including that final LF.
	return lbf.flush(olen, len(p), lbf.indexOfFinalNewline+1)
}
