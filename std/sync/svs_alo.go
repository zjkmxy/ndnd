package sync

import (
	gosync "sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
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
	// errpipe is the channel for delivering errors.
	errpipe chan error
	// publpipe is the channel for delivering new publishers.
	publpipe chan enc.Name
	// stop is the stop signal.
	stop chan struct{}

	// error callback
	onError func(error)
	// publisher callback
	onPublisher func(enc.Name)
}

type SvsAloOpts struct {
	// Name is the name of this instance producer.
	Name enc.Name
	// Svs is the options for the underlying SVS instance.
	Svs SvSyncOpts
	// Snapshot is the snapshot strategy.
	Snapshot Snapshot

	// MaxPipelineSize is the number of objects to fetch
	// concurrently for a single publisher (default 10)
	MaxPipelineSize uint64
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

		outpipe:  make(chan svsPubOut, 256),
		errpipe:  make(chan error, 256),
		publpipe: make(chan enc.Name, 256),
		stop:     make(chan struct{}),

		onError:     func(err error) { log.Error(nil, err.Error()) },
		onPublisher: nil,
	}

	// Default options
	if s.opts.MaxPipelineSize == 0 {
		s.opts.MaxPipelineSize = 10
	}

	// Use null snapshot strategy by default
	if s.opts.Snapshot == nil {
		s.opts.Snapshot = &SnapshotNull{}
	} else {
		s.opts.Snapshot.initialize(s.opts.Name, s.opts.Svs.GroupPrefix)
		s.opts.Snapshot.setCallback(s.snapshotCallback)
	}

	// Initialize the SVS instance.
	s.opts.Svs.OnUpdate = s.onSvsUpdate
	s.svs = NewSvSync(s.opts.Svs)

	// Initialize the state vector with our own state.
	boot := s.svs.GetBootTime()
	seq := s.svs.GetSeqNo(s.opts.Name)
	s.state.Set(s.opts.Name.TlvStr(), boot, svsDataState{
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

// SetOnError sets the error callback.
// You can likely cast the received error as SyncError.
//
// SyncError includes the name of the affected publisher and the error.
// Applications can use this callback to selectively unsubscribe from
// publishers that are not responding.
func (s *SvsALO) SetOnError(callback func(error)) {
	s.onError = callback
}

// SetOnPublisher sets the publisher callback.
//
// This will be called when an update from a new publisher is received.
// This includes both updates for publishers that are already subscribed
// and other non-subscribed publishers. Applications can use this callback
// to test the liveness of publishers and selectively subscribe to them.
func (s *SvsALO) SetOnPublisher(callback func(enc.Name)) {
	s.onPublisher = callback
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
			s.consumeCheck(node, node.TlvStr())
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
// Only this thread has interaction with the application.
func (s *SvsALO) run() {
	defer s.svs.Stop()
	for {
		select {
		case <-s.stop:
			return
		case out := <-s.outpipe:
			s.deliver(out)
		case err := <-s.errpipe:
			s.onError(err)
		case publ := <-s.publpipe:
			s.onPublisher(publ)
		}
	}
}

// onSvsUpdate is the handler for new sequence numbers.
func (s *SvsALO) onSvsUpdate(update SvSyncUpdate) {
	// App callback for new publishers
	if s.onPublisher != nil {
		s.publpipe <- update.Name
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Update the latest known state.
	hash := update.Name.TlvStr()
	entry := s.state.Get(hash, update.Boot)
	entry.Latest = update.High
	s.state.Set(hash, update.Boot, entry)

	// Check if we want to queue new fetch for this update.
	s.consumeCheck(update.Name, hash)
}

// deliver is the outgoing pipeline handler on the run() thread.
func (s *SvsALO) deliver(out svsPubOut) {
	for _, sub := range out.subs {
		sub(out.pub)
	}

	// Commit the successful delivery
	if out.snapstate != nil {
		for _, svEntry := range out.snapstate.Entries {
			hash := svEntry.Name.TlvStr()
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
