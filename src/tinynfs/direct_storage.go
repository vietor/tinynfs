package tinynfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type DirectStorage struct {
	root string
}

func (self *DirectStorage) Close() {
}

func (self *DirectStorage) ReadFile(filepath string) ([]byte, error) {
	return ioutil.ReadFile(self.root + "/" + filepath)
}

func (self *DirectStorage) WriteFile(extname string, data []byte) (string, error) {
	randText := RandHex(5)
	pathText := randText[0:2] + "/" + randText[2:4]
	nameText := randText[5:] + fmt.Sprintf("%x", time.Now().UnixNano())

	if err := os.MkdirAll((self.root + "/" + pathText), 0777); err != nil {
		return "", err
	}
	filepath := pathText + "/" + nameText
	if len(extname) > 0 {
		filepath = filepath + "." + extname
	}
	if err := ioutil.WriteFile((self.root + "/" + filepath), data, 0644); err != nil {
		return "", err
	}
	return filepath, nil
}

func NewDirectStorage(root string) (*DirectStorage, error) {
	if err := os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}
	storage := &DirectStorage{
		root: root,
	}
	return storage, nil
}
