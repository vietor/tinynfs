package tinynfs

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var myRand = struct {
	lock sync.Mutex
	rand *mrand.Rand
}{
	rand: mrand.New(mrand.NewSource(time.Now().UnixNano())),
}

func RandHex(bytes int) (hex string) {
	randBytes := make([]byte, bytes)
	if _, err := io.ReadFull(crand.Reader, randBytes); err != nil {
		myRand.lock.Lock()
		myRand.rand.Read(randBytes)
		myRand.lock.Unlock()
	}
	return fmt.Sprintf("%x", randBytes)
}

type FileLock struct {
	file     *os.File
	lockfile string
}

func (self *FileLock) Lock() error {
	if err := os.MkdirAll(filepath.Dir(self.lockfile), 0777); err != nil {
		return err
	}
	file, err := os.OpenFile(self.lockfile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	err = SysFlock(int(file.Fd()))
	if err != nil {
		file.Close()
		return errors.New("File already locked: " + self.lockfile)
	}
	file.Truncate(0)
	file.Write([]byte(fmt.Sprintf("%d", os.Getpid())))
	self.file = file
	return nil
}

func (self *FileLock) Unlock() {
	if self.file != nil {
		SysUnflock(int(self.file.Fd()))
		self.file.Close()
		self.file = nil
	}
}

func NewFileLock(filepath string) *FileLock {
	return &FileLock{
		lockfile: filepath,
	}
}
