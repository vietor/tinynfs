package tinynfs

import (
	crand "crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"sync"
	"time"
)

var (
	utilRand      = mrand.New(mrand.NewSource(time.Now().UnixNano()))
	utilRandMutex sync.Mutex
)

func randHex(bytes int) (hex string) {
	randBytes := make([]byte, bytes)
	if _, err := io.ReadFull(crand.Reader, randBytes); err != nil {
		utilRandMutex.Lock()
		utilRand.Read(randBytes)
		utilRandMutex.Unlock()
	}
	return fmt.Sprintf("%x", randBytes)
}
