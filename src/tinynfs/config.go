package tinynfs

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type Network struct {
	Tcp                 string
	FileBind            string
	ImageBind           string
	ImageFilePath       string
	ImageOtimizeSize    int
	ImageOtimizeSide    int
	ImageThumbnailSizes map[string]bool
}

type VolumeGroup struct {
	Id   int
	Path string
}

type Storage struct {
	DiskRemain       int64
	SnapshotInterval int64
	SnapshotReserve  int
	VolumeSliceSize  int64
	VolumeFileGroups []VolumeGroup
}

type Config struct {
	Network *Network
	Storage *Storage
}

func (self *Config) Dump() string {
	lines := []string{
		"# configuration",
	}
	lines = append(lines, "network.tcp="+self.Network.Tcp)
	lines = append(lines, "network.file.bind="+self.Network.FileBind)
	lines = append(lines, "network.image.bind="+self.Network.ImageBind)
	lines = append(lines, "network.image.path="+self.Network.ImageFilePath)
	sizes := make([]string, 0, len(self.Network.ImageThumbnailSizes))
	for k := range self.Network.ImageThumbnailSizes {
		sizes = append(sizes, k)
	}
	lines = append(lines, fmt.Sprintf("network.image.optimize.size=%d #Bytes", self.Network.ImageOtimizeSize))
	lines = append(lines, fmt.Sprintf("network.image.optimize.side=%d", self.Network.ImageOtimizeSide))
	lines = append(lines, "network.image.thumbnail.sizes="+strings.Join(sizes, ","))
	lines = append(lines, fmt.Sprintf("storage.disk.remain=%d #Bytes", self.Storage.DiskRemain))
	lines = append(lines, fmt.Sprintf("storage.snapshot.interval=%d #Seconds", self.Storage.SnapshotInterval))
	lines = append(lines, fmt.Sprintf("storage.snapshot.reserve=%d", self.Storage.SnapshotReserve))
	lines = append(lines, fmt.Sprintf("storage.volume.slicesize=%d #Bytes", self.Storage.VolumeSliceSize))
	for _, v := range self.Storage.VolumeFileGroups {
		lines = append(lines, fmt.Sprintf("storage.volume.filegroups=%d:%s", v.Id, v.Path))
	}
	return strings.Join(lines, "\n")
}

func parseBytes(s string) (uint64, error) {
	if m, _ := regexp.MatchString("^[0-9]+(M|m|G|g|K|k)?(B|b)?$", s); !m {
		return 0, ErrParam
	}

	nn := func(c rune) bool {
		return !unicode.IsDigit(c)
	}
	pos := strings.IndexFunc(s, nn)
	if pos == -1 {
		return strconv.ParseUint(s, 10, 64)
	}

	value, err := strconv.ParseUint(s[:pos], 10, 64)
	if err != nil {
		return 0, err
	}

	var bytes uint64
	unit := strings.ToUpper(s[pos:])
	if len(unit) > 0 {
		switch unit[:1] {
		case "G":
			bytes = uint64(value * 1024 * 1024 * 1024)
		case "M":
			bytes = uint64(value * 1024 * 1024)
		case "K":
			bytes = uint64(value * 1024)
		case "B":
			bytes = uint64(value * 1)
		}
	}

	return bytes, nil
}

func NewConfig(filepath string) (*Config, error) {
	config := &Config{
		Network: &Network{
			Tcp:              "tcp4",
			FileBind:         ":7119",
			ImageBind:        ":7120",
			ImageFilePath:    "/image1/",
			ImageOtimizeSize: 100 * 1024,
			ImageOtimizeSide: 2048,
			ImageThumbnailSizes: map[string]bool{
				"192x192": true,
				"240x240": true,
			},
		},
		Storage: &Storage{
			DiskRemain:       100 * 1024 * 1024,
			SnapshotInterval: 1800,
			SnapshotReserve:  2,
			VolumeSliceSize:  5 * 1024 * 1024 * 1024,
			VolumeFileGroups: []VolumeGroup{
				VolumeGroup{
					Id:   0,
					Path: "{{DATA}}/volumes/",
				},
			},
		},
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	no := 0
	vfgs := []VolumeGroup{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		no = no + 1
		line := strings.TrimSpace(scanner.Text())
		if len(line) < 1 || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.Split(line, "#")[0])
		fields := strings.Split(line, "=")
		if len(fields) != 2 {
			return nil, fmt.Errorf("line %d: muiltiple `=`", no)
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		if len(key) < 1 || len(value) < 1 {
			return nil, fmt.Errorf("line %d: empty key or value", no)
		}
		switch key {
		case "network.tcp":
			if m, _ := regexp.MatchString("^(tcp|tcp4|tcp6)$", value); !m {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Network.Tcp = value
			}
		case "network.file.bind":
			if m, _ := regexp.MatchString("^[:0-9a-zA-Z]*:[0-9]+$", value); !m {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Network.FileBind = value
			}
		case "network.image.bind":
			if m, _ := regexp.MatchString("^[:0-9a-zA-Z]*:[0-9]+$", value); !m {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Network.ImageBind = value
			}
		case "network.image.path":
			if m, _ := regexp.MatchString("^\\/[^\\ ]+\\/*$", value); !m {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Network.ImageFilePath = value
			}
		case "network.image.optimize.size":
			size, err := parseBytes(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Network.ImageOtimizeSize = int(size)
			}
		case "network.image.optimize.side":
			side, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Network.ImageOtimizeSide = int(side)
			}
		case "network.image.thumbnail.sizes":
			if m, _ := regexp.MatchString("^[0-9x,]+$", value); !m {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Network.ImageThumbnailSizes = map[string]bool{}
				for _, v := range strings.Split(value, ",") {
					if len(v) > 0 {
						config.Network.ImageThumbnailSizes[v] = true
					}
				}
			}
		case "storage.disk.remain":
			size, err := parseBytes(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Storage.DiskRemain = int64(size)
			}
		case "storage.snapshot.interval":
			count, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Storage.SnapshotInterval = int64(count)
			}
		case "storage.snapshot.reserve":
			count, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Storage.SnapshotReserve = int(count)
			}
		case "storage.volume.slicesize":
			size, err := parseBytes(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				config.Storage.VolumeSliceSize = int64(size)
			}
		case "storage.volume.filegroups":
			if m, _ := regexp.MatchString("^[0-9]{1,2}:\\/.*\\/+$", value); !m {
				return nil, fmt.Errorf("line %d: %s", no, err)
			} else {
				fields := strings.Split(value, ":")
				id, _ := strconv.ParseUint(fields[0], 10, 32)
				vfgs = append(vfgs, VolumeGroup{
					Id:   int(id),
					Path: fields[1],
				})
			}
		default:
			fmt.Printf("ignore line: %d: %s\n", no, line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("ignore error:", err)
	}
	if len(vfgs) > 0 {
		config.Storage.VolumeFileGroups = vfgs
	}

	return config, nil
}
