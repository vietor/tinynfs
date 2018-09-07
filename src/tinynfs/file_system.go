package tinynfs

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	bolt "github.com/etcd-io/bbolt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type HashNode struct {
	Size         int   `json:"size"`
	GroupId      int   `json:"group_id"`
	VolumeId     int64 `json:"volume_id"`
	VolumeOffset int64 `json:"volume_offset"`
}

type FileNode struct {
	HashNode
	Mime     string `json:"mime"`
	Metadata string `json:"metadata"`
}

type FileSystem struct {
	root           string
	config         *Storage
	storageDB      *bolt.DB
	timeOnUpdate   int64
	timeOnSnapshot int64
	volumeGroupIds []int
	volumeStorages map[int]*VolumeStorage
}

type WriteOptions struct {
	Overwrite bool
}

var (
	fileBucket          = []byte("files")
	hashBucket          = []byte("hashs")
	defaultWriteOptions = &WriteOptions{
		Overwrite: true,
	}
)

func (self *FileSystem) init() error {
	db, err := bolt.Open(filepath.Join(self.root, "storage.db"), 0644, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		if err == bolt.ErrTimeout {
			err = ErrFileSystemBusy
		}
		return err
	}
	for _, v := range self.config.VolumeFileGroups {
		volumepath := strings.Replace(v.Path, "{{DATA}}", self.root, 1)
		vs, err := NewVolumeStorage(volumepath, self.config.VolumeSliceSize, self.config.DiskRemain)
		if err != nil {
			db.Close()
			for _, v := range self.volumeStorages {
				v.Close()
			}
			return err
		}
		self.volumeStorages[v.Id] = vs
		self.volumeGroupIds = append(self.volumeGroupIds, v.Id)
	}
	self.storageDB = db
	self.storageDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(fileBucket)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	self.storageDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(hashBucket)
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
	for _, v := range self.volumeStorages {
		v.Close()
	}
}

func (self *FileSystem) readNode(bucket []byte, key []byte, node interface{}) error {
	return self.storageDB.View(func(tx *bolt.Tx) error {
		bt := tx.Bucket(bucket)
		v := bt.Get(key)
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, node)
	})
}

func (self *FileSystem) writeNode(bucket []byte, key []byte, node interface{}) error {
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
	var fnode *FileNode
	if err := self.readNode(fileBucket, []byte(filepath), &fnode); err != nil {
		return "", "", nil, err
	}
	if fnode == nil {
		return "", "", nil, ErrNotExist
	}
	volumeStorage := self.volumeStorages[fnode.GroupId]
	if volumeStorage == nil {
		return "", "", nil, ErrNotExist
	}
	data, err := volumeStorage.ReadFile(fnode.VolumeId, fnode.VolumeOffset, fnode.Size)
	if err != nil {
		return "", "", nil, ErrNotExist
	}
	return fnode.Mime, fnode.Metadata, data, nil
}

func (self *FileSystem) WriteFile(filepath string, filemime string, metadata string, data []byte, options *WriteOptions) error {
	if options == nil {
		options = defaultWriteOptions
	}

	dstat, err := GetPathDiskStat(self.root)
	if err != nil {
		return err
	} else if dstat.Free < uint64(self.config.DiskRemain) {
		return ErrFileSystemFully
	}

	filekey := []byte(filepath)
	if !options.Overwrite {
		var fnode *FileNode
		if err := self.readNode(fileBucket, filekey, &fnode); err != nil {
			return err
		}
		if fnode != nil {
			return ErrExist
		}
	}

	var (
		hnode *HashNode
		fnode *FileNode
	)
	hashtmp := sha256.Sum256(data)
	hashkey := hashtmp[:]
	if err := self.readNode(hashBucket, hashkey, &hnode); err != nil {
		return err
	}
	if hnode != nil {
		fnode = &FileNode{*hnode, filemime, metadata}
	} else {
		// CONFUSED: leak node when same hash concurrent write
		var (
			groupId       int
			volumeStorage *VolumeStorage
		)
		for _, id := range self.volumeGroupIds {
			storage := self.volumeStorages[id]
			if f, _ := storage.IsFully(); !f {
				groupId = id
				volumeStorage = storage
				break
			}
		}
		if volumeStorage == nil {
			return ErrVolumeStorageFully
		}
		volumeId, volumeOffset, err := volumeStorage.WriteFile(data)
		if err != nil {
			return err
		}
		hnode = &HashNode{len(data), groupId, volumeId, volumeOffset}
		if err = self.writeNode(hashBucket, hashkey, hnode); err != nil {
			return err
		}
		self.timeOnUpdate = time.Now().Unix()
		fnode = &FileNode{*hnode, filemime, metadata}
	}

	err = self.writeNode(fileBucket, filekey, fnode)
	if err == nil {
		self.timeOnUpdate = time.Now().Unix()
	}
	return err
}

func (self *FileSystem) DeleteFile(filepath string) error {
	filekey := []byte(filepath)
	var fnode *FileNode
	if err := self.readNode(fileBucket, filekey, &fnode); err != nil {
		return err
	}
	if fnode == nil {
		return ErrNotExist
	}
	err := self.storageDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(fileBucket)
		return bt.Delete(filekey)
	})
	if err == nil {
		self.timeOnUpdate = time.Now().Unix()
	}
	return err
}

func (self *FileSystem) Snapshot(force bool) (string, error) {
	if !force {
		if self.timeOnSnapshot >= self.timeOnUpdate {
			return "", nil
		}
		if self.timeOnSnapshot+self.config.SnapshotInterval > time.Now().Unix() {
			return "", nil
		}
	}
	sspath := filepath.Join(self.root, "snapshots")
	if err := os.MkdirAll(sspath, 0777); err != nil {
		return "", err
	}
	// Get needless snapshot names
	ssfiles, err := ioutil.ReadDir(sspath)
	if err != nil {
		return "", err
	}
	ssnames := []string{}
	for _, file := range ssfiles {
		name := file.Name()
		if m, _ := regexp.MatchString("^storage\\.db\\.", name); m {
			ssnames = append(ssnames, name)
		}
	}
	if len(ssnames) > self.config.SnapshotReserve {
		sort.Strings(ssnames)
		ssnames = ssnames[:len(ssnames)-self.config.SnapshotReserve]
	}
	// Create new snapshot
	uptime := self.timeOnUpdate
	ssname := fmt.Sprintf("storage.db.%d.gz", time.Now().UnixNano())
	gzfile, err := os.OpenFile(filepath.Join(sspath, ssname), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	writer := gzip.NewWriter(gzfile)
	err = self.storageDB.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(writer)
		return err
	})
	writer.Close()
	gzfile.Close()
	if err != nil {
		os.Remove(filepath.Join(sspath, ssname))
		return "", err
	}
	self.timeOnSnapshot = uptime
	// Remove needless snapshot files
	for _, name := range ssnames {
		os.Remove(filepath.Join(sspath, name))
	}

	return ssname, err
}

func NewFileSystem(root string, config *Storage) (*FileSystem, error) {
	if err := os.MkdirAll(root, 0777); err != nil {
		return nil, err
	}

	uptime := time.Now().Unix()
	fs := &FileSystem{
		root:           root,
		config:         config,
		timeOnUpdate:   uptime,
		timeOnSnapshot: uptime,
		volumeGroupIds: []int{},
		volumeStorages: map[int]*VolumeStorage{},
	}
	if err := fs.init(); err != nil {
		return nil, err
	}
	return fs, nil
}
