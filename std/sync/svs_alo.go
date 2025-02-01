package sync

import (
	"slices"
	gosync "sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// SvsALO is a Sync Transport with At Least One delivery semantics.
type SvsALO struct {
	// opts is the configuration options.
	opts SvsAloOpts
	// svs is the underlying Sync Transport.
	svs *SvSync
	// client is the object client.
	client ndn.Client

	// mutex protects the instance.
	mutex gosync.Mutex

	// state is the current state.
	state SvMap[svsDataState]
	// delivered is the delivered state vector.
	// owned by the run() thread (no-lock)
	delivered SvMap[uint64]
	// nodePs is the Pub/Sub coordinator for publisher prefixes
	nodePs SimplePs[SvsPub]

	// outpipe is the channel for delivering data.
	outpipe chan svsPubOut
	// stop is the stop signal.
	stop chan struct{}
}

type SvsAloOpts struct {
	// Name is the name of this instance producer.
	Name enc.Name
	// Svs is the options for the underlying SVS instance.
	Svs SvSyncOpts
	// Snapshot is the snapshot strategy.
	Snapshot Snapshot
}

// NewSvsALO creates a new SvsALO instance.
func NewSvsALO(opts SvsAloOpts) *SvsALO {
	if len(opts.Name) == 0 {
		panic("Name is required")
	}

	s := &SvsALO{
		opts:   opts,
		svs:    nil,
		client: opts.Svs.Client,

		mutex: gosync.Mutex{},

		state:     NewSvMap[svsDataState](0),
		delivered: NewSvMap[uint64](0),
		nodePs:    NewSimplePs[SvsPub](),

		outpipe: make(chan svsPubOut, 256),
		stop:    make(chan struct{}),
	}

	// Use default snapshot strategy if not provided.
	if s.opts.Snapshot == nil {
		s.opts.Snapshot = &SnapshotNull{}
	} else {
		s.opts.Snapshot.setNames(s.opts.Name, s.opts.Svs.GroupPrefix)
		s.opts.Snapshot.setCallback(s.snapshotCallback)
	}

	// Initialize the SVS instance.
	s.opts.Svs.OnUpdate = s.onSvsUpdate
	s.svs = NewSvSync(s.opts.Svs)

	// Initialize the state vector with our own state.
	boot := s.svs.GetBootTime()
	seq := s.svs.GetSeqNo(s.opts.Name)
	s.state.Set(s.opts.Name.String(), boot, svsDataState{
		Known:   seq,
		Latest:  seq,
		Pending: seq,
	})

	return s
}

// String is the log identifier.
func (s *SvsALO) String() string {
	return "svs-alo"
}

// Start starts the SvsALO instance.
func (s *SvsALO) Start() error {
	if err := s.svs.Start(); err != nil {
		return err
	}
	go s.run()
	return nil
}

// Stop stops the SvsALO instance.
func (s *SvsALO) Stop() {
	s.stop <- struct{}{}
	close(s.stop)
}

// Publish sends a message to the group
func (s *SvsALO) Publish(content enc.Wire) (enc.Name, error) {
	return s.produceObject(content)
}

// SubscribePublisher subscribes to all publishers matchin a name prefix.
// Only one subscriber per prefix is allowed.
// If the prefix is already subscribed, the callback is replaced.
func (s *SvsALO) SubscribePublisher(prefix enc.Name, callback func(SvsPub)) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.nodePs.Subscribe(prefix, callback)
	if err != nil {
		return err
	}

	// Trigger a fetch for all known producers matching this prefix.
	for node := range s.state.Iter() {
		if prefix.IsPrefix(node) {
			s.consumeCheck(node, node.String())
		}
	}

	return nil
}

// UnsubscribePublisher unsubscribes removes callbacks added with subscribe.
// The callback may still receive messages for some time after this call.
// The application must handle these messages correctly.
func (s *SvsALO) UnsubscribePublisher(prefix enc.Name) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.nodePs.Unsubscribe(prefix)
}

// run is the main loop for the SvsALO instance.
func (s *SvsALO) run() {
	defer s.svs.Stop()
	for {
		select {
		case <-s.stop:
			return
		case out := <-s.outpipe:
			s.deliver(out)
		}
	}
}

// onSvsUpdate is the handler for new sequence numbers.
func (s *SvsALO) onSvsUpdate(update SvSyncUpdate) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update the latest known state.
	hash := update.Name.String()
	entry := s.state.Get(hash, update.Boot)
	entry.Latest = update.High
	s.state.Set(hash, update.Boot, entry)

	// Check if we want to queue new fetch for this update.
	s.consumeCheck(update.Name, hash)
}

func (s *SvsALO) deliver(out svsPubOut) {
	for _, sub := range out.subs {
		sub(out.pub)
	}

	// Commit the successful delivery
	if out.snapstate != nil {
		for _, svEntry := range out.snapstate.Entries {
			hash := svEntry.Name.String()
			for _, seqEntry := range svEntry.SeqNoEntries {
				prev := s.delivered.Get(hash, seqEntry.BootstrapTime)
				if seqEntry.SeqNo > prev {
					s.delivered.Set(hash, seqEntry.BootstrapTime, seqEntry.SeqNo)
				}
			}
		}
	} else {
		prev := s.delivered.Get(out.hash, out.pub.BootTime)
		if out.pub.SeqNum > prev {
			s.delivered.Set(out.hash, out.pub.BootTime, out.pub.SeqNum)
		}
	}
}

func (s *SvsALO) snapshotCallback(callback snapshotCallbackInner) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	snapPub, ok := callback(s.state)
	if !ok {
		return
	}

	// Update delivered vector in order
	out := svsPubOut{
		pub:       snapPub,
		subs:      slices.Collect(s.nodePs.Subs(snapPub.Publisher)), // suspicious
		snapstate: s.state.Encode(func(state svsDataState) uint64 { return state.Known }),
	}
	s.outpipe <- out

	// Trigger fetch for all affected publishers
	for _, svEntry := range out.snapstate.Entries {
		if snapPub.Publisher.IsPrefix(svEntry.Name) {
			s.consumeCheck(svEntry.Name, svEntry.Name.String())
		}
	}
}
