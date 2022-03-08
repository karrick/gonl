package gonl

import (
	"errors"
	"io"
)

// LineTerminatedReader reads from the source io.Reader and ensures
// the final byte read from it is a newline.
type LineTerminatedReader struct {
	R                   io.Reader
	savedErr            error
	wasFinalByteNewline bool
}

// Read reads up to len(p) bytes into p. It returns the number of
// bytes read (0 <= n <= len(p)) and any error encountered.
func (r *LineTerminatedReader) Read(p []byte) (int, error) {
	var err error
	var n int

	if r.savedErr != nil {
		// We only get here after this io.Reader has received an EOF
		// from the underlying io.Reader.
		if len(p) == 0 {
			return 0, nil // from io.Reader documentation
		}

		// NOTE: io.Reader documentation allows returning some bytes
		// read with the terminating EOF.
		p[0] = '\n'

		// Return the exact error that the underlying io.Reader
		// provided to this.
		err = r.savedErr
		r.savedErr = nil
		return 1, err
	}

	n, err = r.R.Read(p)
	if n > 0 {
		// Only update final byte was newline if at least one byte.
		r.wasFinalByteNewline = p[n-1] == '\n'
	}

	if r.wasFinalByteNewline || err == nil || !errors.Is(err, io.EOF) {
		return n, err
	}
	// POST: Received EOF but final byte was not a newline.

	if n < len(p) {
		// Provided buffer can accommodate the final newline.
		p[n] = '\n'
		return n + 1, err
	}

	// No room to append newline to this buffer. Return the newline
	// and this error next time method is invoked.
	r.savedErr = err
	return n, nil
}
