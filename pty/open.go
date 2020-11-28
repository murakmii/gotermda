package pty

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func Open() (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}

	slave, err := openSlave(master)
	if err != nil {
		master.Close()
		return nil, nil, err
	}

	return master, slave, nil
}

func openSlave(master *os.File) (*os.File, error) {
	var n uint32
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); err != 0 {
		return nil, err
	}

	var unlock uint32
	if _, _, err := syscall.Syscall(syscall.SYS_IOCTL, master.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); err != 0 {
		return nil, err
	}

	return os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
}
