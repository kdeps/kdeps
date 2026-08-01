//go:build !windows

package logging

import "os"

// enableVirtualTerminal is a no-op on platforms whose terminals already
// interpret ANSI escape sequences natively.
func enableVirtualTerminal(_ *os.File) bool { return true }
