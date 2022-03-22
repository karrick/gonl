package gonl

import (
	"crypto/hmac"
	"crypto/sha256"
	"hash"
	"io"
)

// discardWriteCloser is an io.WriteCloser that simply tracks how many
// bytes have been written to it. Used in tests to provide a place to
// write data to, and ensure expected number of bytes have been
// written to it.
type discardWriteCloser struct {
	count int
}

func (dw *discardWriteCloser) Close() error {
	return nil
}

func (dw *discardWriteCloser) Write(p []byte) (int, error) {
	dw.count += len(p)
	return len(p), nil
}

// testBuffer is an io.WriteCloser, and io.Reader, that maintains the
// contents of the data written to it. Used in tests to be able to
// spot check the contents of what has been written to it. Only reason
// this is defined is because bytes.Buffer is not also an io.Closer.
type testBuffer struct {
	slice []byte
}

func (b *testBuffer) Bytes() []byte { return b.slice }

func (b *testBuffer) Close() error { return nil }

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

func (b *testBuffer) String() string { return string(b.slice) }

func (b *testBuffer) Write(buf []byte) (int, error) {
	b.slice = append(b.slice, buf...)
	return len(buf), nil
}

// hashWriteCloser is an io.WriteCloser that does some work with the
// data it is being given, namely writing it to a hash. Used in tests
// to be able to verify the exact data was written to it after all the
// writing has completed, and to provide an expected penalty for each
// Write call on the io.Writer when streaming data.
type hashWriteCloser struct {
	mac hash.Hash
}

func newHashWriteCloser(key []byte) *hashWriteCloser {
	return &hashWriteCloser{mac: hmac.New(sha256.New, key)}
}

func (dw *hashWriteCloser) Close() error {
	return nil
}

func (dw *hashWriteCloser) MAC() []byte {
	return dw.mac.Sum(nil)
}

func (dw *hashWriteCloser) Reset() {
	dw.mac.Reset()
}

func (dw *hashWriteCloser) ValidMAC(want []byte) bool {
	return hmac.Equal(want, dw.mac.Sum(nil))
}

func (dw *hashWriteCloser) Write(p []byte) (int, error) {
	return dw.mac.Write(p)
}
