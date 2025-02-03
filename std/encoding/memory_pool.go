package encoding

import (
	"bytes"
	"hash"
	"sync"

	"github.com/cespare/xxhash"
)

type hashPoolObj struct {
	hash   hash.Hash64
	buffer bytes.Buffer
}

var xxHashPool = sync.Pool{
	New: func() any { return xxHashPoolNew() },
}

func xxHashPoolNew() *hashPoolObj {
	return &hashPoolObj{
		hash:   xxhash.New(),
		buffer: bytes.Buffer{},
	}
}

func xxHashPoolGet() *hashPoolObj {
	obj := xxHashPool.Get().(*hashPoolObj)
	obj.hash.Reset()
	obj.buffer.Reset()
	return obj
}

func xxHashPoolPut(obj *hashPoolObj) {
	xxHashPool.Put(obj)
}
