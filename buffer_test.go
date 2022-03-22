package gonl

import (
	"crypto/hmac"
	"crypto/sha256"
	"hash"
	"io"
)

// dumpWriteCloser is an io.WriteCloser that simply tracks how many
// bytes have been written to it. Used in tests to provide a place to
// write data to, and ensure expected number of bytes have been
// written to it.
type dumpWriteCloser struct {
	count int
}

func (dw *dumpWriteCloser) Close() error {
	return nil
}

func (dw *dumpWriteCloser) Write(p []byte) (int, error) {
	dw.count += len(p)
	return len(p), nil
}

// testBuffer is an io.Closer, io.Reader and io.Writer that maintains
// the contents of the data written to it. Used in tests to be able to
// spot check the contents of what has been written to it.
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

// workWriteCloser is an io.WriteCloser that does some work with the
// data it is being given, namely writing it to a hash. Used in tests
// to be able to verify the exact data was written to it after all the
// writing has completed.
type workWriteCloser struct {
	mac hash.Hash
}

func newWorkWriteCloser(key []byte) *workWriteCloser {
	return &workWriteCloser{mac: hmac.New(sha256.New, key)}
}

func (dw *workWriteCloser) Close() error {
	return nil
}

func (dw *workWriteCloser) MAC() []byte {
	return dw.mac.Sum(nil)
}

func (dw *workWriteCloser) Reset() {
	dw.mac.Reset()
}

func (dw *workWriteCloser) ValidMAC(want []byte) bool {
	return hmac.Equal(want, dw.mac.Sum(nil))
}

func (dw *workWriteCloser) Write(p []byte) (int, error) {
	return dw.mac.Write(p)
}
