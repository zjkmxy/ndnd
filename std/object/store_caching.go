package object

import (
	"sync"
	"time"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
)

// CachingStore is a store that caches objects from another store.
type CachingStore struct {
	// store is the underlying store.
	store ndn.Store
	// cache is the cache.
	cache *MemoryStore
	// journal is the journal.
	journal []storeJournalEntry
	// flushDelay is the flush delay.
	flushDelay time.Duration
	// jmutex is the journal mutex.
	jmutex sync.Mutex
	// txp is the active transaction parent
	txp *CachingStore
	// flushTimer is the flush timer.
	flushTimer *time.Timer
}

type storeJournalEntry struct {
	// name is the name of the object.
	name enc.Name
	// version is the version of the object.
	version uint64
	// wire is the wire of the object.
	wire []byte
	// action is the action.
	action storeJournalAction
}

type storeJournalAction uint8

const (
	storeJournalActionPut = iota
	storeJournalActionRemove
)

// NewCachingStore creates a new caching store.
func NewCachingStore(store ndn.Store, flushDelay time.Duration) *CachingStore {
	return &CachingStore{
		store:      store,
		cache:      NewMemoryStore(),
		journal:    make([]storeJournalEntry, 0),
		flushDelay: flushDelay,
	}
}

func (s *CachingStore) Close() error {
	return s.Flush()
}

func (s *CachingStore) String() string {
	return "caching-store"
}

func (s *CachingStore) Get(name enc.Name, prefix bool) ([]byte, error) {
	wire, err := s.cache.Get(name, prefix)
	if wire != nil || err != nil {
		return wire, err
	}
	return s.store.Get(name, prefix)
}

func (s *CachingStore) Put(name enc.Name, version uint64, wire []byte) error {
	if err := s.cache.Put(name, version, wire); err != nil {
		return err
	}

	s.jmutex.Lock()
	defer s.jmutex.Unlock()

	s.journal = append(s.journal, storeJournalEntry{
		name:    name,
		version: version,
		wire:    wire,
		action:  storeJournalActionPut,
	})

	s.scheduleFlush()

	return nil
}

func (s *CachingStore) Remove(name enc.Name, prefix bool) error {
	if err := s.cache.Remove(name, prefix); err != nil {
		return err
	}

	s.jmutex.Lock()
	defer s.jmutex.Unlock()

	s.journal = append(s.journal, storeJournalEntry{
		name:   name,
		action: storeJournalActionRemove,
	})

	s.scheduleFlush()

	return nil
}

func (s *CachingStore) Begin() (ndn.Store, error) {
	cacheTxn, err := s.cache.Begin()
	if err != nil {
		return nil, err
	}

	return &CachingStore{
		store:   s.store,
		cache:   cacheTxn.(*MemoryStore),
		journal: make([]storeJournalEntry, 0),
		txp:     s,
	}, nil
}

func (s *CachingStore) Commit() error {
	if err := s.cache.Commit(); err != nil {
		return err
	}

	s.txp.jmutex.Lock()
	defer s.txp.jmutex.Unlock()

	s.txp.journal = append(s.txp.journal, s.journal...)
	s.txp.scheduleFlush()

	return nil
}

func (s *CachingStore) Rollback() error {
	return s.cache.Rollback()
}

// Flush flushes the cache.
func (s *CachingStore) Flush() error {
	s.jmutex.Lock()
	journal := s.journal
	s.journal = s.journal[len(journal):]
	s.jmutex.Unlock()

	if len(journal) == 0 {
		return nil
	}

	if err := func() error {
		tx, err := s.store.Begin()
		if err != nil {
			return err
		}
		defer tx.Commit()

		for _, entry := range journal {
			switch entry.action {
			case storeJournalActionPut:
				if err := tx.Put(entry.name, entry.version, entry.wire); err != nil {
					return err
				}
			case storeJournalActionRemove:
				if err := tx.Remove(entry.name, false); err != nil {
					return err
				}
			}
		}

		return nil
	}(); err != nil {
		return err
	}

	// Evict cache entries from PUT journal entries.
	// TODO: this is incorrect if a new PUT entry is added during the flush.
	for _, entry := range journal {
		if entry.action == storeJournalActionPut {
			s.cache.Remove(entry.name, false)
		}
	}

	return nil
}

func (s *CachingStore) scheduleFlush() {
	if s.flushTimer != nil || s.txp != nil || len(s.journal) == 0 {
		return
	}

	// scheduleFlush is called with mutex held.
	// If we're doing an immediate flush, release the mutex first.
	if s.flushDelay == 0 {
		s.jmutex.Unlock()
		defer s.jmutex.Lock()

		if err := s.Flush(); err != nil {
			log.Error(s, "Failed to flush cache", "err", err)
		}
		return
	}

	// Otherwise, schedule a flush.
	s.flushTimer = time.AfterFunc(s.flushDelay, func() {
		if err := s.Flush(); err != nil {
			log.Error(s, "Failed to flush cache", "err", err)
		}

		s.jmutex.Lock()
		defer s.jmutex.Unlock()

		s.flushTimer = nil
		s.scheduleFlush()
	})
}
