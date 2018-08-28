package tinynfs

import (
	crand "crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"sync"
	"time"
)

var myRand = struct {
	lock sync.Mutex
	rand *mrand.Rand
}{
	rand: mrand.New(mrand.NewSource(time.Now().UnixNano())),
}

func RandHex(bytes int) (hex string) {
	randBytes := make([]byte, bytes)
	if _, err := io.ReadFull(crand.Reader, randBytes); err != nil {
		myRand.lock.Lock()
		myRand.rand.Read(randBytes)
		myRand.lock.Unlock()
	}
	return fmt.Sprintf("%x", randBytes)
}
