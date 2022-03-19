//go:build !gonl_debug
// +build !gonl_debug

package gonl

// debug is a no-op for release builds
func debug(_ string, _ ...interface{}) {}
