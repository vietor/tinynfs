package tinynfs

import (
	crand "crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DiskStat struct {
	Size uint64
	Used uint64
	Free uint64
}

var myRand = struct {
	lock sync.Mutex
	rand *mrand.Rand
}{
	rand: mrand.New(mrand.NewSource(time.Now().UnixNano())),
}

func TimeHex(style int) string {
	var ts int64
	if style == 0 {
		ts = time.Now().Unix()
	} else {
		ts = time.Now().UnixNano()
	}
	return fmt.Sprintf("%x", ts)
}

func RandHex(bytes int) string {
	randBytes := make([]byte, bytes)
	if _, err := io.ReadFull(crand.Reader, randBytes); err != nil {
		myRand.lock.Lock()
		myRand.rand.Read(randBytes)
		myRand.lock.Unlock()
	}
	return fmt.Sprintf("%x", randBytes)
}

type ProcessLock struct {
	file     *os.File
	lockfile string
}

func (self *ProcessLock) Lock() error {
	if err := os.MkdirAll(filepath.Dir(self.lockfile), 0777); err != nil {
		return err
	}
	file, err := os.OpenFile(self.lockfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if err = SysFlock(int(file.Fd())); err != nil {
		file.Close()
		return fmt.Errorf("file already locked: %s", self.lockfile)
	}
	file.Truncate(0)
	file.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	self.file = file
	return nil
}

func (self *ProcessLock) Unlock() {
	if self.file != nil {
		SysUnflock(int(self.file.Fd()))
		self.file.Close()
		self.file = nil
	}
}

func NewProcessLock(filepath string) *ProcessLock {
	return &ProcessLock{
		lockfile: filepath,
	}
}
