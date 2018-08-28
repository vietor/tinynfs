package tinynfs

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

func NewConfig(filename string) *Config {
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
	return config
}
