package tinynfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type VolumeFile struct {
	id    int64
	size  int64
	rFile *os.File
	wFile *os.File
	wLock sync.Mutex
}

type VolumeStorage struct {
	root       string
	limit      int64
	volumes    map[int64]*VolumeFile
	volumeMap  map[int64]*VolumeFile
	volumeLock sync.Mutex
}

func (self *VolumeStorage) init() (err error) {
	files, err := ioutil.ReadDir(self.root)
	if err != nil {
		return err
	}
	for _, file := range files {
		name := file.Name()
		if m, _ := regexp.MatchString("^volume-[0-9]+$", name); m {
			id, err := strconv.ParseInt(name[7:], 10, 64)
			if err == nil {
				v, err := self.mkVolumeFile(id, file.Size())
				if err == nil {
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

func (self *VolumeStorage) getFreeVolume() (*VolumeFile, error) {
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

func (self *VolumeStorage) ReadFile(id int64, offset int64, size int) (data []byte, err error) {
	self.volumeLock.Lock()
	v := self.volumeMap[id]
	self.volumeLock.Unlock()
	if v == nil {
		return nil, os.ErrNotExist
	}

	data = make([]byte, size)
	_, err = v.rFile.ReadAt(data, offset)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (self *VolumeStorage) WriteFile(data []byte) (id int64, offset int64, err error) {
	v, err := self.getFreeVolume()
	if err != nil {
		return 0, 0, err
	}

	v.wLock.Lock()
	defer v.wLock.Unlock()

	offset = v.size
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

func NewVolumeStorage(root string, limit int64) (storage *VolumeStorage, err error) {
	if err = os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}

	storage = &VolumeStorage{
		root:      root,
		limit:     limit,
		volumes:   map[int64]*VolumeFile{},
		volumeMap: map[int64]*VolumeFile{},
	}
	if err = storage.init(); err != nil {
		return nil, err
	}
	return storage, nil
}
