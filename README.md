# gonl

Go newline library provides a small collection of functions and data
structures useful for stream processing text on newline boundaries.

## Features

### BatchLineWriter

BatchLineWriter is an io.WriteCloser that buffers output to ensure it
only emits bytes to the underlying io.WriteCloser on line feed
boundaries.

It is important for caller to Close the BatchLineWriter to flush any
residual data that was not terminated with a newline.

Compare this structure with PerLineWriter. This structure is not
suitable for situations that require line buffering. This structure is
used to reduce the number of Write invocations on the underlying
io.WriteCloser by buffering data, but calling its Write method only
invokes Write on the underlying io.WriteCloser with a newline
terminated sequence of bytes, potentially with more than one line
being written at a time.

### LineTerminatedReader

LineTerminatedReader reads from the source io.Reader and ensures the
final byte read from it is a newline.

### NewlineCounter

NewlineCounter counts the number of lines from the io.Reader until it
receives a read error, such as io.EOF, and returns the number of lines
read. It will return the same number regardless of whether the final
Read terminated in a newline character or not.

### OneNewline

OneNewline returns a string with exactly one terminating newline
character. More simple than strings.TrimRight. When input string ends
with multiple OneNewline characters, it will strip off all but first
one, reusing the same underlying string bytes. When string does not
end in a OneNewline character, it returns the original string with a
OneNewline character appended. Newline characters before any
non-OneNewline characters are ignored.

### PerLineWriter

PerLineWriter is a synchronous io.WriteCloser which writes each
completed newline terminated line to the underlying io.WriteCloser.

This stream processor ensures there is exactly one Write call made to
the underlying io.WriteCloser for each newline terminated line being
written to it.

Compare this structure with BatchLineWriter. This structure is
suitable for situations that require line buffering. This structure is
used to ensure each newline terminated line is individually sent to
the underlying io.WriteCloser. Calling its Write method only invokes
Write on the underlying io.WriteCloser with a newline terminated
sequence of bytes.
