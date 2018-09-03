package tinynfs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	VolumeValidateTimestamp = 1530000000000000000
)

type VolumeFile struct {
	id    int64
	size  int64
	rFile *os.File
	wFile *os.File
	wLock sync.Mutex
}

type VolumeStorage struct {
	root        string
	limit       int64
	volumes     map[int64]*VolumeFile
	volumeMap   map[int64]*VolumeFile
	volumeLock  sync.Mutex
	volumePlock *ProcessLock
}

func (self *VolumeStorage) init() error {
	if time.Now().UnixNano() < VolumeValidateTimestamp {
		return ErrTimestamp
	}
	if err := self.volumePlock.Lock(); err != nil {
		return err
	}
	files, err := ioutil.ReadDir(self.root)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := file.Name()
		if m, _ := regexp.MatchString("^volume-[0-9]+$", name); m {
			id, err := strconv.ParseInt(name[7:], 10, 64)
			if err != nil || id < VolumeValidateTimestamp {
				log.Println(fmt.Sprintf("not volume file %s", name))
			} else {
				v, err := self.mkVolumeFile(id, file.Size())
				if err != nil {
					log.Println(fmt.Sprintf("load failed %s %s", name, err))
				} else {
					if v.size < self.limit {
						self.volumes[v.id] = v
					}
					self.volumeMap[v.id] = v
				}
			}
		}
	}
	return nil
}

func (self *VolumeStorage) Close() {
	self.volumeLock.Lock()
	defer self.volumeLock.Unlock()

	for _, v := range self.volumeMap {
		v.rFile.Close()
		v.wFile.Close()
	}
	self.volumePlock.Unlock()
}

func (self *VolumeStorage) mkVolumeFile(id int64, size int64) (*VolumeFile, error) {
	filepath := self.root + fmt.Sprintf("/volume-%d", id)
	w, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	r, err := os.OpenFile(filepath, os.O_RDONLY, 0644)
	if err != nil {
		w.Close()
		return nil, err
	}
	v := &VolumeFile{
		id:    id,
		size:  size,
		rFile: r,
		wFile: w,
	}
	return v, nil
}

func (self *VolumeStorage) mkWriteVolume() (*VolumeFile, error) {
	self.volumeLock.Lock()
	defer self.volumeLock.Unlock()

	for _, v := range self.volumes {
		if v.size < self.limit {
			return v, nil
		}
	}
	v, err := self.mkVolumeFile(time.Now().UnixNano(), 0)
	if err == nil {
		self.volumes[v.id] = v
		self.volumeMap[v.id] = v
	}
	return v, err
}

func (self *VolumeStorage) ReadFile(id int64, offset int64, size int) ([]byte, error) {
	self.volumeLock.Lock()
	v := self.volumeMap[id]
	self.volumeLock.Unlock()
	if v == nil {
		return nil, os.ErrNotExist
	}

	data := make([]byte, size)
	if _, err := v.rFile.ReadAt(data, offset); err != nil {
		return nil, err
	}
	return data, nil
}

func (self *VolumeStorage) WriteFile(data []byte) (int64, int64, error) {
	v, err := self.mkWriteVolume()
	if err != nil {
		return 0, 0, err
	}

	v.wLock.Lock()
	defer v.wLock.Unlock()

	offset := v.size
	n, err := v.wFile.WriteAt(data, offset)
	if err != nil {
		return 0, 0, err
	}
	v.wFile.Sync()
	v.size += int64(n)
	if v.size >= self.limit {
		self.volumeLock.Lock()
		delete(self.volumes, v.id)
		self.volumeLock.Unlock()
	}
	return v.id, offset, nil
}

func NewVolumeStorage(root string, limit int64) (*VolumeStorage, error) {
	if err := os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}

	storage := &VolumeStorage{
		root:        root,
		limit:       limit,
		volumes:     map[int64]*VolumeFile{},
		volumeMap:   map[int64]*VolumeFile{},
		volumePlock: NewProcessLock(root + "/volume.lock"),
	}
	if err := storage.init(); err != nil {
		return nil, err
	}
	return storage, nil
}
