package gonl

import (
	"bytes"
	"errors"
	"io"
)

func (lw *BatchLineWriter) ReadFrom(r io.Reader) (int64, error) {
	var totalRead int64

	for {
		leno := lw.bufferLength()
		m := lw.bufferGrow(minRead)
		lw.buf = lw.buf[:m]

		nr, rerr := r.Read(lw.buf[m:cap(lw.buf)])
		if nr < 0 {
			return totalRead, errors.New("invalid read result")
		}

		lw.buf = lw.buf[:m+nr]

		// NEWLINE LOGIC

		p := lw.buf[m : m+nr]
		if finalIndex := bytes.LastIndexByte(p, '\n'); finalIndex >= 0 {
			lw.indexOfFinalNewline = m + finalIndex
		}

		if lw.bufferLength() >= lw.flushThreshold && lw.indexOfFinalNewline >= 0 {
			// Flush some data
			nw, werr := lw.flush(leno, len(p), lw.indexOfFinalNewline+1)
			if werr != nil {
				return totalRead + int64(nw), werr
			}
		}

		// END OF NEWLINE LOGIC

		totalRead += int64(nr)

		if rerr == io.EOF {
			// NOTE: This does not flush remaining data, because there
			// may be additional bytes to send to line writer.
			return totalRead, nil
		}
		if rerr != nil {
			// NOTE: This does not flush remaining data, because there
			// may be additional bytes to send to line writer.
			return totalRead, rerr
		}
	}
}
