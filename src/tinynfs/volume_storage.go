package tinynfs

import (
	"fmt"
	"os"
)

type VolumeStorage struct {
	fileroot string
}

func (self *VolumeStorage) getFilePath(id int) string {
	return self.fileroot + fmt.Sprintf("/volume-%04d", id)
}

func (self *VolumeStorage) ReadFile(id int, offset int64, size int) (data []byte, err error) {
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

func (self *VolumeStorage) WriteFile(id int, offset int64, data []byte) (err error) {
	f, err := os.OpenFile(self.getFilePath(id), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	defer f.Close()

	_, err = f.WriteAt(data, offset)
	return err
}

func NewVolumeStorage(fileroot string) (storage *VolumeStorage, err error) {
	err = os.MkdirAll(fileroot, 0777)
	if err != nil {
		return nil, err
	}
	storage = &VolumeStorage{fileroot}
	return storage, nil
}
