//go:build !windows

package connect

func enableVirtualTerminalProcessing() {
    // Not needed on Unix systems as they support ANSI escape sequences by default
}
