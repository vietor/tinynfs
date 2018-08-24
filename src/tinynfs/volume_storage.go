package tinynfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"time"
)

type VolumeInfo struct {
	id   int64
	size int64
}

type VolumeStorage struct {
	root        string
	limit       int64
	volumes     map[int64]*VolumeInfo
	fullVolumes map[int64]*VolumeInfo
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
				v := &VolumeInfo{
					id,
					file.Size(),
				}
				if v.size < self.limit {
					self.volumes[v.id] = v
				} else {
					self.fullVolumes[v.id] = v
				}
			}
		}
	}
	return nil
}

func (self *VolumeStorage) getFilePath(id int64) string {
	return self.root + fmt.Sprintf("/volume-%d", id)
}

func (self *VolumeStorage) getFreeVolume() *VolumeInfo {
	for _, v := range self.volumes {
		if v.size < self.limit {
			return v
		}
	}
	v := &VolumeInfo{
		time.Now().UnixNano(),
		0,
	}
	self.volumes[v.id] = v
	return v
}

func (self *VolumeStorage) classifyVolume(v *VolumeInfo) {
	if v.size < self.limit {
		return
	}
	self.fullVolumes[v.id] = v
	delete(self.volumes, v.id)
}

func (self *VolumeStorage) ReadFile(id int64, offset int64, size int) (data []byte, err error) {
	f, err := os.Open(self.getFilePath(id))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data = make([]byte, size)
	_, err = f.ReadAt(data, offset)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (self *VolumeStorage) WriteFile(data []byte) (id int64, offset int64, err error) {
	v := self.getFreeVolume()
	f, err := os.OpenFile(self.getFilePath(v.id), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	offset = v.size
	n, err := f.WriteAt(data, offset)
	if err != nil {
		return 0, 0, err
	}
	v.size += int64(n)
	self.classifyVolume(v)
	return v.id, offset, nil
}

func NewVolumeStorage(root string, limit int64) (storage *VolumeStorage, err error) {
	err = os.MkdirAll(root, 0777)
	if err != nil {
		return nil, err
	}

	storage = &VolumeStorage{
		root,
		limit,
		map[int64]*VolumeInfo{},
		map[int64]*VolumeInfo{},
	}
	storage.init()
	return storage, nil
}
