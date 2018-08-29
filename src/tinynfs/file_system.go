package tinynfs

import (
	"encoding/json"
	"fmt"
	bolt "github.com/etcd-io/bbolt"
	"os"
	"path/filepath"
	"time"
)

var (
	fileBucket       = []byte("files")
	deleteFileBucket = []byte("deletefiles")
)

type FileNode struct {
	Size         int    `json:"size"`
	Mime         string `json:"mime"`
	Metadata     string `json:"metadata"`
	Storage      int    `json:"storage"` // 0-direct, 1-volume
	DirectFile   string `json:"direct_file"`
	VolumeId     int64  `json:"volume_id"`
	VolumeOffset int64  `json:"volume_offset"`
}

type FileSystem struct {
	root          string
	config        *Storage
	storageDB     *bolt.DB
	directStorage *DirectStorage
	volumeStroage *VolumeStorage
}

func (self *FileSystem) init() error {
	db, err := bolt.Open(filepath.Join(self.root, "storage.db"), 0644, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}
	ds, err := NewDirectStorage(filepath.Join(self.root, "directs"))
	if err != nil {
		db.Close()
		return err
	}
	vs, err := NewVolumeStorage(filepath.Join(self.root, "volumes"), self.config.VolumeMaxSize)
	if err != nil {
		db.Close()
		ds.Close()
		return err
	}
	self.storageDB = db
	self.directStorage = ds
	self.volumeStroage = vs
	self.storageDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket(fileBucket)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	self.storageDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket(deleteFileBucket)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return nil
}

func (self *FileSystem) Close() {
	if self.storageDB != nil {
		self.storageDB.Close()
	}
	if self.directStorage != nil {
		self.directStorage.Close()
	}
	if self.volumeStroage != nil {
		self.volumeStroage.Close()
	}
}

func (self *FileSystem) getFileNode(bucket []byte, key []byte) (*FileNode, error) {
	var node *FileNode
	err := self.storageDB.View(func(tx *bolt.Tx) error {
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
	return self.storageDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(bucket)
		b, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return bt.Put(key, b)
	})
}

func (self *FileSystem) ReadFile(filepath string) (string, string, []byte, error) {
	node, _ := self.getFileNode(fileBucket, []byte(filepath))
	if node == nil {
		return "", "", nil, os.ErrNotExist
	}
	var (
		data []byte
		err  error
	)
	if node.Storage == 0 {
		data, err = self.directStorage.ReadFile(node.DirectFile)
	} else {
		data, err = self.volumeStroage.ReadFile(node.VolumeId, node.VolumeOffset, node.Size)
	}
	return node.Mime, node.Metadata, data, err
}

func (self *FileSystem) WriteFile(filepath string, filemime string, metadata string, data []byte) error {
	dstat, err := GetDiskStat(self.root)
	if err != nil {
		return err
	} else if dstat.Free < uint64(self.config.DiskRemain) {
		return ErrDiskFull
	}

	oldnode, _ := self.getFileNode(fileBucket, []byte(filepath))
	var (
		node *FileNode
		size = len(data)
	)
	if size > int(self.config.DirectMinSize) {
		directpath, err := self.directStorage.WriteFile("", data)
		if err != nil {
			return err
		}
		node = &FileNode{size, filemime, metadata, 0, directpath, 0, 0}
	} else {
		volumeId, volumeOffset, err := self.volumeStroage.WriteFile(data)
		if err != nil {
			return err
		}
		node = &FileNode{size, filemime, metadata, 1, "", volumeId, volumeOffset}
	}
	err = self.putFileNode(fileBucket, []byte(filepath), node)
	if err == nil && oldnode != nil {
		self.putFileNode(deleteFileBucket, []byte(fmt.Sprintf("%s\r\n%d", filepath, time.Now().UnixNano())), oldnode)
	}
	return err
}

func (self *FileSystem) DeleteFile(filepath string) error {
	node, _ := self.getFileNode(fileBucket, []byte(filepath))
	if node == nil {
		return os.ErrNotExist
	}

	err := self.storageDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(fileBucket)
		return bt.Delete([]byte(filepath))
	})
	if err == nil {
		self.putFileNode(deleteFileBucket, []byte(fmt.Sprintf("%s\r\n%d", filepath, time.Now().UnixNano())), node)
	}
	return err
}

func NewFileSystem(root string, config *Storage) (*FileSystem, error) {
	if err := os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}

	fs := &FileSystem{
		root:   root,
		config: config,
	}
	if err := fs.init(); err != nil {
		return nil, err
	}
	return fs, nil
}
