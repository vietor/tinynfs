package tinynfs

import (
	"io/ioutil"
	"os"
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
	randtext := RandHex(5)
	subpath := randtext[0:2] + "/" + randtext[2:4]
	filename := randtext[5:] + TimeHex(1)
	if len(extname) > 0 {
		filename = filename + "." + extname
	}

	if err := os.MkdirAll((self.root + "/" + subpath), 0777); err != nil {
		return "", err
	}

	filepath := subpath + "/" + filename
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
