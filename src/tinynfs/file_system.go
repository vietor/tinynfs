package tinynfs

import (
	"encoding/json"
	"fmt"
	bolt "github.com/coreos/bbolt"
	"os"
	"path/filepath"
	"time"
)

var (
	FileBucket = []byte("files")
)

type FileNode struct {
	Size         int    `json:"size"`
	Mime         string `json:"mime"`
	Storage      int    `json:"storage"` // 0-direct, 1-volume
	DirectFile   string `json:"direct_file"`
	VolumeId     int64  `json:"volume_id"`
	VolumeOffset int64  `json:"volume_offset"`
}

type FileSystem struct {
	root          string
	config        *Storage
	directoryDB   *bolt.DB
	directStorage *DirectStorage
	volumeStroage *VolumeStorage
}

func (self *FileSystem) init() (err error) {
	if self.directoryDB, err = bolt.Open(filepath.Join(self.root, "directory.db"), 0644, &bolt.Options{Timeout: 1 * time.Second}); err != nil {
		return err
	}
	if self.directStorage, err = NewDirectStorage(filepath.Join(self.root, "directs")); err != nil {
		return err
	}
	if self.volumeStroage, err = NewVolumeStorage(filepath.Join(self.root, "volumes"), self.config.VolumeLimit); err != nil {
		return err
	}
	self.directoryDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket(FileBucket)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return nil
}

func (self *FileSystem) getFileNode(bucket []byte, key []byte) (*FileNode, error) {
	var node *FileNode
	err := self.directoryDB.View(func(tx *bolt.Tx) error {
		bt := tx.Bucket(bucket)
		v := bt.Get(key)
		if v != nil {
			return json.Unmarshal(v, &node)
		}
		return nil
	})
	if err != nil {
		node = nil
	}
	return node, err
}

func (self *FileSystem) putFileNode(bucket []byte, key []byte, node *FileNode) error {
	return self.directoryDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(bucket)
		b, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return bt.Put(key, b)
	})
}

func (self *FileSystem) ReadFile(filepath string) (filemime string, data []byte, err error) {
	node, _ := self.getFileNode(FileBucket, []byte(filepath))
	if node == nil {
		return "", nil, os.ErrNotExist
	}
	if node.Storage == 0 {
		data, err = self.directStorage.ReadFile(node.DirectFile)
	} else {
		data, err = self.volumeStroage.ReadFile(node.VolumeId, node.VolumeOffset, node.Size)
	}
	return node.Mime, data, err
}

func (self *FileSystem) WriteFile(filepath string, filemime string, data []byte) (err error) {
	var node *FileNode

	if node == nil {
		size := len(data)
		if size > int(self.config.DirectLimit) {
			directpath, err := self.directStorage.WriteFile("", data)
			if err != nil {
				return err
			}
			node = &FileNode{size, filemime, 0, directpath, 0, 0}
		} else {
			volumeId, volumeOffset, err := self.volumeStroage.WriteFile(data)
			if err != nil {
				return err
			}
			node = &FileNode{size, filemime, 1, "", volumeId, volumeOffset}
		}
	}
	return self.directoryDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(FileBucket)
		b, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return bt.Put([]byte(filepath), b)
	})
}

func (self *FileSystem) DeleteFile(filepath string) (err error) {
	node, _ := self.getFileNode(FileBucket, []byte(filepath))
	if node == nil {
		return os.ErrNotExist
	}

	return self.directoryDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(FileBucket)
		return bt.Delete([]byte(filepath))
	})
}

func NewFileSystem(root string, config *Storage) (fs *FileSystem, err error) {
	if err = os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}

	fs = &FileSystem{
		root:   root,
		config: config,
	}
	if err = fs.init(); err != nil {
		return nil, err
	}
	return fs, nil
}
