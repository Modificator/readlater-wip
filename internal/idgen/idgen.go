package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"sync/atomic"
	"time"
)

var fallbackCounter uint64

func New(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err == nil {
		return prefix + "_" + hex.EncodeToString(buf)
	}
	seq := atomic.AddUint64(&fallbackCounter, 1)
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000000") + "_" + strconv.FormatUint(seq, 10)
}
