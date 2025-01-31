package sync

import (
	"sort"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	spec_svs "github.com/named-data/ndnd/std/ndn/svs/v3"
)

// Map representation of the state vector.
type SvMap[V any] map[string][]svMapVal[V]

// One entry in the state vector map.
type svMapVal[V any] struct {
	boot  uint64
	value V
}

// Create a new state vector map.
func NewSvMap[V any](size int) SvMap[V] {
	return make(SvMap[V], size)
}

// Get seq entry for a bootstrap time.
func (m SvMap[V]) Get(hash string, boot uint64) (value V) {
	// TODO: binary search - this is sorted
	for _, entry := range m[hash] {
		if entry.boot == boot {
			return entry.value
		}
	}
	return value
}

func (m SvMap[V]) Set(hash string, boot uint64, value V) {
	for i, entry := range m[hash] {
		if entry.boot == boot {
			m[hash][i].value = value
			return
		}
	}

	m[hash] = append(m[hash], svMapVal[V]{boot, value})
	sort.Slice(m[hash], func(i, j int) bool {
		return m[hash][i].boot < m[hash][j].boot
	})
}

// Check if a SvMap is newer than another.
// cmp(a, b) is the function to compare values (a > b).
func (m SvMap[V]) IsNewerThan(other SvMap[V], cmp func(V, V) bool) bool {
	// TODO: optimize with two pointers
	for hash, entries := range m {
		for _, entry := range entries {
			foundOther := false
			for _, otherEntry := range other[hash] {
				if otherEntry.boot == entry.boot {
					foundOther = true
					if cmp(entry.value, otherEntry.value) {
						return true
					}
				}
			}
			if !foundOther {
				return true
			}
		}
	}
	return false
}

// Encode the state vector map to a StateVector.
// seq is the function to get the sequence number
func (m SvMap[V]) Encode(seq func(V) uint64) *spec_svs.StateVector {
	sv := &spec_svs.StateVector{
		Entries: make([]*spec_svs.StateVectorEntry, 0, len(m)),
	}

	for hash, vals := range m {
		name, err := enc.NameFromStr(hash)
		if err != nil {
			log.Error(nil, "Invalid name in SV map", "hash", hash)
			continue
		}

		entry := &spec_svs.StateVectorEntry{
			Name:         name,
			SeqNoEntries: make([]*spec_svs.SeqNoEntry, 0, len(vals)),
		}
		sv.Entries = append(sv.Entries, entry)

		for _, val := range vals {
			if seqNo := seq(val.value); seqNo > 0 {
				entry.SeqNoEntries = append(entry.SeqNoEntries, &spec_svs.SeqNoEntry{
					BootstrapTime: val.boot,
					SeqNo:         seqNo,
				})
			}
		}
	}

	// Sort entries by in the NDN canonical order
	sort.Slice(sv.Entries, func(i, j int) bool {
		return sv.Entries[i].Name.Compare(sv.Entries[j].Name) < 0
	})

	return sv
}
