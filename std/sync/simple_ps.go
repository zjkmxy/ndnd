package sync

import (
	gosync "sync"

	enc "github.com/named-data/ndnd/std/encoding"
)

// SimplePs is a simple Pub/Sub system.
type SimplePs[V any] struct {
	// mutex protects the instance.
	mutex gosync.RWMutex
	// subs is the list of subscribers.
	subs map[string]SimplePsSub[V]
}

type SimplePsSub[V any] struct {
	// Prefix is the name prefix to subscribe.
	Prefix enc.Name
	// Callback is the callback function.
	Callback func(V)
}

func NewSimplePs[V any]() SimplePs[V] {
	return SimplePs[V]{
		subs: make(map[string]SimplePsSub[V]),
	}
}

func (ps *SimplePs[V]) Subscribe(prefix enc.Name, callback func(V)) error {
	if callback == nil {
		panic("Callback is required for subscription")
	}

	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.subs[prefix.String()] = SimplePsSub[V]{
		Prefix:   prefix,
		Callback: callback,
	}

	return nil
}

func (ps *SimplePs[V]) Unsubscribe(prefix enc.Name) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	delete(ps.subs, prefix.String())
}

func (ps *SimplePs[V]) Publish(name enc.Name, data V) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	for _, sub := range ps.subs {
		if sub.Prefix.IsPrefix(name) {
			sub.Callback(data)
		}
	}
}
