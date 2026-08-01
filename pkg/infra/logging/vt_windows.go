//go:build windows

package logging

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVirtualTerminal turns on ANSI escape-sequence interpretation for f's
// console handle. Classic Windows consoles print raw escape bytes literally
// unless ENABLE_VIRTUAL_TERMINAL_PROCESSING is set on the handle. Returns
// false when f isn't a console or the OS predates VT support (pre-Windows 10
// TH2), in which case the caller must disable colors instead of emitting
// codes the terminal can't interpret.
func enableVirtualTerminal(f *os.File) bool {
	handle := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false
	}
	if mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0 {
		return true
	}
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	return windows.SetConsoleMode(handle, mode) == nil
}
