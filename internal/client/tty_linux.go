//go:build linux

package client

import (
	"os"
	"syscall"
	"unsafe"
)

func isTerminal() bool {
	var term struct {
		Iflag, Oflag, Cflag, Lflag uint32
		Line                       uint8
		CC                         [32]uint8
		Ispeed, Ospeed             uint32
	}
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&term)))
	return e == 0
}
