package tinynfs

import (
	"compress/gzip"
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

type FileNode struct {
	Size         int    `json:"size"`
	Mime         string `json:"mime"`
	Metadata     string `json:"metadata"`
	GroupId      int    `json:"group_id"`
	VolumeId     int64  `json:"volume_id"`
	VolumeOffset int64  `json:"volume_offset"`
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
	deleteFileBucket    = []byte("deletefiles")
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
		_, err := tx.CreateBucketIfNotExists(deleteFileBucket)
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

func (self *FileSystem) readNode(bucket []byte, key []byte) (*FileNode, error) {
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

func (self *FileSystem) writeNode(bucket []byte, key []byte, node *FileNode) error {
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
	node, _ := self.readNode(fileBucket, []byte(filepath))
	if node == nil {
		return "", "", nil, ErrNotExist
	}
	volumeStorage := self.volumeStorages[node.GroupId]
	if volumeStorage == nil {
		return "", "", nil, ErrNotExist
	}
	data, err := volumeStorage.ReadFile(node.VolumeId, node.VolumeOffset, node.Size)
	if err != nil {
		return "", "", nil, ErrNotExist
	}
	return node.Mime, node.Metadata, data, nil
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

	oldnode, _ := self.readNode(fileBucket, []byte(filepath))
	if oldnode != nil && !options.Overwrite {
		return ErrExist
	}

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
	node := &FileNode{len(data), filemime, metadata, groupId, volumeId, volumeOffset}

	err = self.writeNode(fileBucket, []byte(filepath), node)
	if err == nil {
		self.timeOnUpdate = time.Now().Unix()
		if oldnode != nil {
			self.writeNode(deleteFileBucket, []byte(fmt.Sprintf("%s\r\n%d", filepath, time.Now().UnixNano())), oldnode)
		}
	}
	return err
}

func (self *FileSystem) DeleteFile(filepath string) error {
	node, _ := self.readNode(fileBucket, []byte(filepath))
	if node == nil {
		return ErrNotExist
	}

	err := self.storageDB.Update(func(tx *bolt.Tx) error {
		bt := tx.Bucket(fileBucket)
		return bt.Delete([]byte(filepath))
	})
	if err == nil {
		self.timeOnUpdate = time.Now().Unix()
		self.writeNode(deleteFileBucket, []byte(fmt.Sprintf("%s\r\n%d", filepath, time.Now().UnixNano())), node)
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
