package tinynfs

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	bolt "github.com/coreos/bbolt"
	"os"
	"path/filepath"
	"time"
)

var (
	HashBucket = []byte("hashs")
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
	enableHash    bool
	directLimit   int64
	volumeLimit   int64
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
	if self.volumeStroage, err = NewVolumeStorage(filepath.Join(self.root, "volumes"), self.volumeLimit); err != nil {
		return err
	}
	self.directoryDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket(FileBucket)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if self.enableHash {
		self.directoryDB.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucket(HashBucket)
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
			return nil
		})
	}
	return nil
}

func (self *FileSystem) ReadFile(filename string) (filemime string, data []byte, err error) {
	var node *FileNode = nil

	self.directoryDB.View(func(tx *bolt.Tx) error {
		bt := tx.Bucket(FileBucket)
		v := bt.Get([]byte(filename))
		if v != nil {
			err := json.Unmarshal(v, &node)
			if err != nil {
				node = nil
			}
		}
		return nil
	})
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

func (self *FileSystem) WriteFile(filename string, filemime string, data []byte) (err error) {
	var node *FileNode = nil

	hash := sha256.Sum256(data)
	if self.enableHash {
		self.directoryDB.View(func(tx *bolt.Tx) error {
			bt := tx.Bucket(HashBucket)
			v := bt.Get(hash[:])
			if v != nil {
				err := json.Unmarshal(v, &node)
				if err != nil {
					node = nil
				}
			}
			return nil
		})
	}
	if node == nil {
		size := len(data)
		if size > int(self.directLimit) {
			directname, err := self.directStorage.WriteFile("", data)
			if err != nil {
				return err
			}
			node = &FileNode{size, filemime, 0, directname, 0, 0}
		} else {
			volumeId, volumeOffset, err := self.volumeStroage.WriteFile(data)
			if err != nil {
				return err
			}
			node = &FileNode{size, filemime, 1, "", volumeId, volumeOffset}
		}
		if self.enableHash {
			self.directoryDB.Update(func(tx *bolt.Tx) error {
				bt := tx.Bucket(HashBucket)
				b, err := json.Marshal(node)
				if err != nil {
					return err
				}
				return bt.Put(hash[:], b)
			})
		}
	}
	return self.directoryDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(FileBucket)
		b, err := json.Marshal(node)
		if err != nil {
			return err
		}
		return bt.Put([]byte(filename), b)
	})
}

func NewFileSystem(root string) (fs *FileSystem, err error) {
	if err = os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}

	fs = &FileSystem{}
	fs.root = root
	fs.enableHash = true
	fs.directLimit = 4 * 1024 * 1024
	fs.volumeLimit = 4 * 1024 * 1024 * 1024
	if err = fs.init(); err != nil {
		return nil, err
	}
	return fs, nil
}
