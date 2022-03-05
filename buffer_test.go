package gonl

import "io"

type testBuffer struct {
	slice []byte
}

func (b *testBuffer) Bytes() []byte { return b.slice }

func (b *testBuffer) Close() error { return nil }

func (b *testBuffer) String() string { return string(b.slice) }

// Write writes len(p) bytes from p to the Buffer.
func (b *testBuffer) Write(buf []byte) (int, error) {
	b.slice = append(b.slice, buf...)
	return len(buf), nil
}

// Read reads up to len(p) bytes into p from the Buffer.
func (b *testBuffer) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(b.slice) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.slice)
	b.slice = b.slice[n:]
	return n, nil
}
