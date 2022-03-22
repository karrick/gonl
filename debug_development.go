//go:build gonl_debug
// +build gonl_debug

package gonl

import (
	"fmt"
	"os"
)

// debug formats and prints arguments to stderr for development builds
func debug(f string, a ...interface{}) {
	os.Stderr.Write([]byte("gonl: " + fmt.Sprintf(f, a...)))
}
