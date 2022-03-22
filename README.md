# gonl

Go newline library provides a small collection of functions and data
structures useful for stream processing text on newline boundaries.

[![License](https://img.shields.io/badge/License-BSD_2--Clause-orange.svg)](https://opensource.org/licenses/BSD-2-Clause)
[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)
[![GoDoc](https://godoc.org/github.com/karrick/gonl?status.svg)](https://godoc.org/github.com/karrick/gonl)
[![GoReportCard](https://goreportcard.com/badge/github.com/karrick/gonl)](https://goreportcard.com/report/github.com/karrick/gonl)

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

```Go
func ExampleBatchLineWriter() {
	// For bulk streaming cases, recommend one use the same size that
	// io.Copy uses by default. For more interactive cases, use a
	// smaller threshold.
	const threshold = 32 * 1024

	source := bytes.NewReader(novel)

	// bytes.Buffer does not provide a Close method, therefore need to
	// wrap it with a structure that provides a no-op Close method.
	bb := new(bytes.Buffer)
	destination := NopCloseWriter(bb)

	lw, err := gonl.NewBatchLineWriter(destination, threshold)
	if err != nil {
		panic(err)
	}

	_, err = lw.ReadFrom(source)
	if err != nil {
		panic(err)
	}

	if err = lw.Close(); err != nil {
		panic(err)
	}

	fmt.Printf("%d\n", bb.Len())
	// Output: 4039275
}
```

### LineTerminatedReader

LineTerminatedReader reads from the source io.Reader and ensures the
final byte read from it is a newline.

```Go
func ExampleLineTerminatedReader() {
	r := &gonl.LineTerminatedReader{R: strings.NewReader("123\n456")}
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	if got, want := len(buf), 8; got != want {
		fmt.Fprintf(os.Stderr, "GOT: %v; WANT: %v\n", got, want)
		os.Exit(1)
	}
	fmt.Printf("%q\n", buf[len(buf)-1])
	// Output: '\n'
}
```

### NewlineCounter

NewlineCounter counts the number of lines from the io.Reader until it
receives a read error, such as io.EOF, and returns the number of lines
read. It will return the same number regardless of whether the final
Read terminated in a newline character or not.

```Go
func ExampleNewlineCounter() {
	c1, err := gonl.NewlineCounter(strings.NewReader("one\ntwo\nthree\n"))
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(c1)

	c2, err := gonl.NewlineCounter(strings.NewReader("one\ntwo\nthree"))
	if err != nil {
		os.Exit(1)
	}
	fmt.Println(c2)
	// Output:
	// 3
	// 3
}
```

### OneNewline

OneNewline returns a string with exactly one terminating newline
character. More simple than strings.TrimRight. When input string ends
with multiple OneNewline characters, it will strip off all but first
one, reusing the same underlying string bytes. When string does not
end in a OneNewline character, it returns the original string with a
OneNewline character appended. Newline characters before any
non-OneNewline characters are ignored.

```Go
func ExampleOneNewline() {
	fmt.Println(gonl.OneNewline("abc\n\ndef\n\n"))
	// Output:
	// abc
	//
	// def
}

```

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

```Go
func ExamplePerLineWriter() error {
    // Flush completed lines to os.Stdout at least every 512 bytes.
    lw := gonl.PerLineWriter{WC: os.Stdout}

    // Give copy buffer some room.
    _, rerr := io.Copy(lw, os.Stdin)

    // Clean up
    cerr := lw.Close()
    if rerr == nil {
        return cerr
    }
    return rerr
}
```
