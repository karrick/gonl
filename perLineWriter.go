package gonl

import (
	"bytes"
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
	unfinished []byte         // bytes without trailing newline
	wc         io.WriteCloser // where data ultimately written
}

// NewPerLineWriter returns a new PerLineWriter that individually
// writes each newline terminated line to the provided
// io.WriteCloser. When a PerLineWriter is closed, it flushes any
// remaining bytes, then closes the provided io.WriteCloser. BUFSIZE
// may be zero, but this requires an unsigned integer in order to
// prevent need to check for negative argument value, and potentially
// having to return an error, and clients having to check an error
// return value.
func NewPerLineWriter(wc io.WriteCloser, bufsize uint) *PerLineWriter {
	lw := &PerLineWriter{wc: wc}

	if bufsize > 0 {
		lw.unfinished = make([]byte, 0, int(bufsize))
	}

	return lw
}

// Close will transform then write any data remaining in the
// PerLineWriter that was not newline terminated, then closes the
// underlying io.WriteCloser.
func (lw *PerLineWriter) Close() error {
	var err error

	if len(lw.unfinished) > 0 {
		// When additional bytes are available to be written, flush
		// them without a newline before we close the stream.
		if _, err = lw.wc.Write(lw.unfinished); err != nil {
			_ = lw.wc.Close()
			return err
		}
		lw.unfinished = lw.unfinished[:0]
	}
	if err = lw.wc.Close(); err != nil {
		return err
	}
	return nil
}

// Write invokes Write on the underlying io.WriteCloser for each
// newline terminated sequence of bytes in p.
func (lw *PerLineWriter) Write(p []byte) (int, error) {
	var fullLine []byte // fullLine is a newline terminated line
	var err error
	var index int
	var nc int // nc is the count of bytes consumed from buf

	for {
		index = bytes.IndexByte(p, '\n')
		if index == -1 {
			// When buf does not contain a newline, we will store
			// whatever bytes left in provided buf in the LineWriter
			// instance, and use it to prefix what we are given next
			// time Write is invoked.
			lw.unfinished = append(lw.unfinished, p...)
			nc += len(p) // pretend like we processed the extra bytes
			return nc, nil
		}
		// POST: buf[index] is a newline.
		index++

		// When a previous Write invocation left bytes without a
		// newline, use it to prefix whatever bytes provided during
		// this invocation.
		if len(lw.unfinished) > 0 {
			// Append new line to whatever was already accumulated
			// from previous invocation.
			fullLine = append(lw.unfinished, p[:index]...)

			// Mark accumulated as consumed, while reusing previously
			// allocated memory pointed to by byte slice.
			lw.unfinished = lw.unfinished[:0]
		} else {
			fullLine = p[:index]
		}

		if _, err = lw.wc.Write(fullLine); err != nil {
			return nc, err
		}

		nc += index   // update number of bytes processed
		p = p[index:] // advance buf to consume bytes processed

		if len(p) == 0 {
			return nc, nil // we are at end
		}
	}
}
