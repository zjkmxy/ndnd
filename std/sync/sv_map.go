package sync

import (
	"sort"

	spec_svs "github.com/named-data/ndnd/std/ndn/svs/v3"
)

// Map representation of the state vector.
type svMap map[uint64][]spec_svs.SeqNoEntry

// Get seq entry for a bootstrap time.
func (m svMap) get(hash uint64, btime uint64) spec_svs.SeqNoEntry {
	for _, entry := range m[hash] {
		if entry.BootstrapTime == btime {
			return entry
		}
	}
	return spec_svs.SeqNoEntry{
		BootstrapTime: btime,
		SeqNo:         0,
	}
}

// Set seq entry for a bootstrap time.
func (m svMap) set(hash uint64, btime uint64, seq uint64) {
	for i, entry := range m[hash] {
		if entry.BootstrapTime == btime {
			m[hash][i].SeqNo = seq
			return
		}
	}
	m[hash] = append(m[hash], spec_svs.SeqNoEntry{
		BootstrapTime: btime,
		SeqNo:         seq,
	})
	sort.Slice(m[hash], func(i, j int) bool {
		return m[hash][i].BootstrapTime < m[hash][j].BootstrapTime
	})
}

// Check if a svHashMap is newer than another.
// If existOnly is true, only check if the other has all entries.
func (m svMap) isNewerThan(other svMap, existOnly bool) bool {
	for hash, entries := range m {
		for _, entry := range entries {
			foundOther := false
			for _, otherEntry := range other[hash] {
				if otherEntry.BootstrapTime == entry.BootstrapTime {
					foundOther = true
					if !existOnly && otherEntry.SeqNo < entry.SeqNo {
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
