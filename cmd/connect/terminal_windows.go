//go:build windows

package connect

import (
    "os"
    "golang.org/x/sys/windows"
)

func enableVirtualTerminalProcessing() {
    stdout := windows.Handle(os.Stdout.Fd())
    var mode uint32
    windows.GetConsoleMode(stdout, &mode)
    windows.SetConsoleMode(stdout, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
