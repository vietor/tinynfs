package tinynfs

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

type DirectStorage struct {
	root string
}

func (self *DirectStorage) ReadFile(filename string) (data []byte, err error) {
	return ioutil.ReadFile(self.root + "/" + filename)
}

func (self *DirectStorage) WriteFile(extname string, data []byte) (filename string, err error) {
	randText := fmt.Sprintf("%04x", rand.Intn(65536))
	timeText := fmt.Sprintf("%x", time.Now().UnixNano())

	pathText := randText[0:2] + "/" + randText[2:4]
	err = os.MkdirAll((self.root + "/" + pathText), 0777)
	if err != nil {
		return "", err
	}
	filename = pathText + "/" + timeText
	if len(extname) > 0 {
		filename = filename + "." + extname
	}
	err = ioutil.WriteFile((self.root + "/" + filename), data, 0644)
	if err != nil {
		return "", err
	}
	return filename, nil
}

func NewDirectStorage(root string) (storage *DirectStorage, err error) {
	if err = os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}
	storage = &DirectStorage{root}
	return storage, nil
}
