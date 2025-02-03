package encoding

import (
	"hash"
	"sync"

	"github.com/cespare/xxhash"
)

type hashPoolObj struct {
	hash   hash.Hash64
	buffer []byte
}

var xxHashPool = sync.Pool{
	New: func() any { return xxHashPoolNew() },
}

func xxHashPoolNew() *hashPoolObj {
	return &hashPoolObj{
		hash:   xxhash.New(),
		buffer: make([]byte, 0, 512),
	}
}

func xxHashPoolGet(bufSize int) *hashPoolObj {
	obj := xxHashPool.Get().(*hashPoolObj)
	obj.hash.Reset()
	if cap(obj.buffer) < bufSize {
		obj.buffer = make([]byte, bufSize)
	} else {
		obj.buffer = obj.buffer[0:bufSize]
	}
	return obj
}

func xxHashPoolPut(obj *hashPoolObj) {
	xxHashPool.Put(obj)
}
