package sync

import (
	gosync "sync"

	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/ndn"
)

// Max pending interests for a producer.
const SvsAloMaxPending = 20

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
	// wmutex protects the write (produce) state.
	wmutex gosync.Mutex

	// state is the current state.
	state SvMap[SeqFetchState]
	// nodePs is the Pub/Sub coordinator for publisher prefixes
	nodePs SimplePs[SvsPub]
	// dataPs is the Pub/Sub coordinator for data prefixes
	dataPs SimplePs[SvsPub]

	// outpipe is the channel for delivering data.
	outpipe chan SvsPub
	// stop is the stop signal.
	stop chan struct{}
}

type SvsAloOpts struct {
	// Name is the name of this instance producer.
	Name enc.Name
	// Svs is the options for the underlying SVS instance.
	Svs SvSyncOpts
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

		mutex:  gosync.Mutex{},
		wmutex: gosync.Mutex{},

		state:  NewSvMap[SeqFetchState](0),
		nodePs: NewSimplePs[SvsPub](),
		dataPs: NewSimplePs[SvsPub](),

		outpipe: make(chan SvsPub, 256),
		stop:    make(chan struct{}),
	}

	s.opts.Svs.OnUpdate = s.onSvsUpdate
	s.svs = NewSvSync(s.opts.Svs)

	return s
}

// String is the log identifier.
func (s *SvsALO) String() string {
	return "svs-alo"
}

// Start starts the SvsALO instance.
func (s *SvsALO) Start() {
	s.svs.Start()
	go s.run()
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
	return s.nodePs.Subscribe(prefix, callback)
}

// UnsubscribePublisher unsubscribes removes callbacks added with subscribe.
func (s *SvsALO) UnsubscribePublisher(prefix enc.Name) {
	s.nodePs.Unsubscribe(prefix)
}

// run is the main loop for the SvsALO instance.
func (s *SvsALO) run() {
	defer s.svs.Stop()
	for {
		select {
		case <-s.stop:
			return
		case pub := <-s.outpipe:
			s.nodePs.Publish(pub.Publisher, pub)
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
