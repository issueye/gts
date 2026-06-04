package stdlib

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const enableVirtualTerminalInputFlag = 0x0200

func terminalEnableVirtualTerminalInput(fd int) error {
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return nil
	}
	if mode&enableVirtualTerminalInputFlag != 0 {
		return nil
	}
	if err := windows.SetConsoleMode(handle, mode|enableVirtualTerminalInputFlag); err != nil {
		return fmt.Errorf("enable virtual terminal input: %w", err)
	}
	return nil
}
