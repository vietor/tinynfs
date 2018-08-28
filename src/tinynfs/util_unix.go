package tinynfs

import (
	"syscall"
)

func SysFlock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
}

func SysUnflock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_UN)
}
