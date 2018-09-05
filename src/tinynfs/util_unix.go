// +build !windows

package tinynfs

import (
	"os"
	"os/signal"
	"syscall"
)

func SysFlock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
}

func SysUnflock(fd int) error {
	return syscall.Flock(fd, syscall.LOCK_UN)
}

func GetPathDiskStat(path string) (*DiskStat, error) {
	fs := syscall.Statfs_t{}
	if err := syscall.Statfs(path, &fs); err != nil {
		return nil, err
	}
	info := &DiskStat{
		Size: fs.Blocks * uint64(fs.Bsize),
		Free: fs.Bfree * uint64(fs.Bsize),
	}
	info.Used = info.Size - info.Free
	return info, nil
}

func WaitProcessExit(onexit OnProcessExit) {
	var (
		sc chan os.Signal
		s  os.Signal
	)
	sc = make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP)
	for {
		s = <-sc
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP:
			onexit()
			return
		default:
			return
		}
	}
}
