package gonl

import (
	"bytes"
	"errors"
	"io"
)

// NewlineCounter counts the number of lines from the io.Reader until
// it receives a read error, such as io.EOF, and returns the number of
// lines read. It will return the same number regardless of whether
// the final Read terminated in a newline character or not.
func NewlineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 4096)
	var err error
	var newlines, total, n int
	var isNotFinalNewline bool

	for {
		n, err = r.Read(buf)
		if n > 0 {
			total += n
			isNotFinalNewline = buf[n-1] != '\n'
			var searchOffset int
			for {
				index := bytes.IndexByte(buf[searchOffset:n], '\n')
				if index == -1 {
					break // done counting newlines from this chunk
				}

				// Count this newline.
				newlines++

				// Start next search following this newline.
				searchOffset += index + 1
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil // io.EOF is expected at end of stream
			}
			break // do not try to read more if error
		}
	}

	// Return the same number of lines read regardless of whether the
	// final read terminated in a newline character.
	if isNotFinalNewline {
		newlines++
	} else if total == 1 {
		newlines--
	}
	return newlines, err
}
