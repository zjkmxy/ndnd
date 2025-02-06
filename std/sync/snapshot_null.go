package sync

// SnapshotNull is a non-snapshot strategy.
type SnapshotNull struct {
}

func (s *SnapshotNull) Snapshot() Snapshot {
	return s
}

func (s *SnapshotNull) initialize(snapPsState) {
}

func (s *SnapshotNull) checkFetch(snapCheckArgs) {
}

func (s *SnapshotNull) checkSelf(SvMap[uint64]) {
}
