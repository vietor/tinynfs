package tinynfs

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type Network struct {
	Tcp      string
	FileBind string
}

type Storage struct {
	DiskRemain    int64
	DirectMinSize int64
	VolumeMaxSize int64
}

type Config struct {
	Network *Network
	Storage *Storage
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

func NewConfig(filepath string) *Config {
	config := &Config{
		Network: &Network{
			Tcp:      "tcp4",
			FileBind: ":7119",
		},
		Storage: &Storage{
			DiskRemain:    50 * 1024 * 1024,
			DirectMinSize: 5 * 1024 * 1024,
			VolumeMaxSize: 5 * 1024 * 1024 * 1024,
		},
	}

	file, err := os.Open(filepath)
	if err != nil {
		log.Println(err)
		return config
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) < 1 || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.Split(line, "#")[0])
		fields := strings.Split(line, "=")
		if len(fields) != 2 {
			log.Println("Bad config line: " + line)
			continue
		}
		key := strings.TrimSpace(fields[0])
		value := strings.TrimSpace(fields[1])
		if len(key) < 1 || len(value) < 1 {
			log.Println("Bad config line: " + line)
			continue
		}
		switch key {
		case "network.tcp":
			if m, _ := regexp.MatchString("^(tcp|tcp4|tcp6)$", value); !m {
				log.Println("Ignore config line:" + line)
			} else {
				config.Network.Tcp = value
			}
		case "network.file.bind":
			if m, _ := regexp.MatchString("^[:0-9a-zA-Z]*:[0-9]+$", value); !m {
				log.Println("Ignore config line:" + line)
			} else {
				config.Network.FileBind = value
			}
		case "storage.disk.remain":
			size, err := parseBytes(value)
			if err != nil {
				log.Println("Ignore config line:" + line)
			} else {
				config.Storage.DiskRemain = int64(size)
			}
		case "storage.direct.minsize":
			size, err := parseBytes(value)
			if err != nil {
				log.Println("Ignore config line:" + line)
			} else {
				config.Storage.DirectMinSize = int64(size)
			}
		case "storage.volume.maxsize":
			size, err := parseBytes(value)
			if err != nil {
				log.Println("Ignore config line:" + line)
			} else {
				config.Storage.VolumeMaxSize = int64(size)
			}
		default:
			log.Println("Ignore config line:" + line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

	return config
}
