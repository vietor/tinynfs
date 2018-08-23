package tinynfs

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

type DirectStorage struct {
	fileroot string
}

func (self *DirectStorage) ReadFile(filename string) (data []byte, err error) {
	return ioutil.ReadFile(self.fileroot + "/" + filename)
}

func (self *DirectStorage) WriteFile(data []byte) (filename string, err error) {
	randText := fmt.Sprintf("%04x", rand.Intn(65536))
	timeText := fmt.Sprintf("%x", time.Now().UnixNano())

	pathText := randText[0:2] + "/" + randText[2:4]
	err = os.MkdirAll((self.fileroot + "/" + pathText), 0777)
	if err != nil {
		return "", err
	}
	filename = pathText + "/" + timeText
	err = ioutil.WriteFile((self.fileroot + "/" + filename), data, 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func NewDirectStorage(fileroot string) (storage *DirectStorage, err error) {
	err = os.MkdirAll(fileroot, 0777)
	if err != nil {
		return nil, err
	}
	storage = &DirectStorage{fileroot}
	return storage, nil
}
