package ssepool

import (
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	defaultBufSize         = 1024
	defaultConnTTL         = 3 * time.Hour
	defaultCleanupInterval = 24 * time.Hour
)

type Pool struct {
	threads sync.Map // key: threadID(string), val: *thread
	bufSize int
}

var defaultPool atomic.Pointer[Pool]

func init() {
	defaultPool.Store(New(defaultBufSize))

}

func Default() *Pool {
	return defaultPool.Load()
}

func New(bufSize int) *Pool {
	if bufSize <= 0 {
		bufSize = defaultBufSize
	}
	newPool := &Pool{bufSize: bufSize}
	defaultPool.Store(newPool)
	return newPool
}

func Attach(threadID string, w http.ResponseWriter) (*Connection, error) {
	return Default().Attach(threadID, w)
}
