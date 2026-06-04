//go:build !windows

package stdlib

func terminalEnableVirtualTerminalInput(fd int) error {
	return nil
}
