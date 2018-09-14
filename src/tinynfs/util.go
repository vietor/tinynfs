package tinynfs

import (
	"fmt"
	"os"
	"path/filepath"
)

type DiskStat struct {
	Size uint64
	Used uint64
	Free uint64
}

type OnProcessExit func()

type ProcessLock struct {
	file     *os.File
	lockfile string
}

func (self *ProcessLock) Lock() (bool, error) {
	if err := os.MkdirAll(filepath.Dir(self.lockfile), 0777); err != nil {
		return false, err
	}
	file, err := os.OpenFile(self.lockfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	if err = SysFlock(int(file.Fd())); err != nil {
		file.Close()
		return false, nil
	}
	file.Truncate(0)
	file.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	file.Sync()
	self.file = file
	return true, nil
}

func (self *ProcessLock) Unlock() bool {
	if self.file != nil {
		SysUnflock(int(self.file.Fd()))
		self.file.Close()
		self.file = nil
		return true
	}
	return false
}

func NewProcessLock(filepath string) *ProcessLock {
	return &ProcessLock{
		lockfile: filepath,
	}
}
