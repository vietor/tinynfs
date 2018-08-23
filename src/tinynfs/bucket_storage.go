package tinynfs

import (
	"os"
	"strconv"
)

type BucketStorage struct {
	fileroot string
}

func (self *BucketStorage) getFilePath(id int) string {
	return self.fileroot + "/volume-" + strconv.Itoa(id)
}

func (self *BucketStorage) ReadFile(id int, offset int64, size int) (data []byte, err error) {
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

func (self *BucketStorage) WriteFile(id int, offset int64, data []byte) (err error) {
	err = os.MkdirAll(self.fileroot, 0777)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(self.getFilePath(id), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil
	}
	defer f.Close()

	_, err = f.WriteAt(data, offset)
	return err
}

func NewBucketStorage(fileroot string) *BucketStorage {
	return &BucketStorage{fileroot}
}
