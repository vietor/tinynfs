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
	Port int
}

type Storage struct {
	EnableHash  bool
	DirectLimit int64
	VolumeLimit int64
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
			Port: 7119,
		},
		Storage: &Storage{
			EnableHash:  true,
			DirectLimit: 4 * 1024 * 1024,
			VolumeLimit: 4 * 1024 * 1024 * 1024,
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
		case "network.port":
			port, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				log.Println("Ignore config line:" + line)
			} else {
				config.Network.Port = int(port)
			}
		case "storage.enablehash":
			enable, err := strconv.ParseBool(value)
			if err != nil {
				log.Println("Ignore config line:" + line)
			} else {
				config.Storage.EnableHash = enable
			}
		case "storage.directlimit":
			size, err := parseBytes(value)
			if err != nil {
				log.Println("Ignore config line:" + line)
			} else {
				config.Storage.DirectLimit = int64(size)
			}
		case "storage.volumelimit":
			size, err := parseBytes(value)
			if err != nil {
				log.Println("Ignore config line:" + line)
			} else {
				config.Storage.VolumeLimit = int64(size)
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
