package arc

import "sync"

// ArcPool is a sync pool based on reference counting.
// When the counter reaches zero, the value is returned to the pool.
type ArcPool[T any] struct {
	reset func(T)
	pool  sync.Pool
}

// NewArcPool creates a new ArcPool[T].
func NewArcPool[T any](new_ func() T, reset func(T)) *ArcPool[T] {
	pool := &ArcPool[T]{}
	pool.reset = reset
	pool.pool = sync.Pool{
		New: func() any { return &Arc[T]{v: new_(), p: pool} },
	}
	return pool
}

// Get returns a new Arc[T] from the pool.
func (p *ArcPool[T]) Get() *Arc[T] {
	val := p.pool.Get().(*Arc[T])
	p.reset(val.Load())
	return val
}
