package dv

import (
	"sync"
	"time"

	"github.com/named-data/ndnd/dv/config"
	"github.com/named-data/ndnd/dv/nfdc"
	"github.com/named-data/ndnd/dv/table"
	enc "github.com/named-data/ndnd/std/encoding"
	"github.com/named-data/ndnd/std/log"
	"github.com/named-data/ndnd/std/ndn"
	mgmt "github.com/named-data/ndnd/std/ndn/mgmt_2022"
	"github.com/named-data/ndnd/std/object"
	sec "github.com/named-data/ndnd/std/security"
	"github.com/named-data/ndnd/std/security/keychain"
	"github.com/named-data/ndnd/std/security/trust_schema"
	ndn_sync "github.com/named-data/ndnd/std/sync"
	"github.com/named-data/ndnd/std/utils"
)

type Router struct {
	// go-ndn app that this router is attached to
	engine ndn.Engine
	// config for this router
	config *config.Config
	// trust configuration
	trust *sec.TrustConfig
	// object client
	client ndn.Client
	// nfd management thread
	nfdc *nfdc.NfdMgmtThread
	// single mutex for all operations
	mutex sync.Mutex

	// channel to stop the DV
	stop chan bool
	// heartbeat for outgoing Advertisements
	heartbeat *time.Ticker
	// deadcheck for neighbors
	deadcheck *time.Ticker

	// neighbor table
	neighbors *table.NeighborTable
	// routing information base
	rib *table.Rib
	// prefix table
	pfx *table.PrefixTable
	// forwarding table
	fib *table.Fib

	// advertisement module
	advert advertModule
	// prefix table svs instance
	pfxSvs *ndn_sync.SvSync
}

// Create a new DV router.
func NewRouter(config *config.Config, engine ndn.Engine) (*Router, error) {
	// Validate configuration
	err := config.Parse()
	if err != nil {
		return nil, err
	}

	// Create packet store
	store := object.NewMemoryStore()

	// Create security configuration
	var trust *sec.TrustConfig = nil
	if config.KeyChainUri == "insecure" {
		log.Warn(nil, "Security is disabled - insecure mode")
	} else {
		kc, err := keychain.NewKeyChain(config.KeyChainUri, store)
		if err != nil {
			return nil, err
		}
		schema, err := trust_schema.NewLvsSchema(config.SchemaBytes())
		if err != nil {
			return nil, err
		}
		anchors := config.TrustAnchorNames()
		trust, err = sec.NewTrustConfig(kc, schema, anchors)
		if err != nil {
			return nil, err
		}
	}

	// Create the DV router
	dv := &Router{
		engine: engine,
		config: config,
		trust:  trust,
		client: object.NewClient(engine, store, trust),
		nfdc:   nfdc.NewNfdMgmtThread(engine),
		mutex:  sync.Mutex{},
	}

	// Initialize advertisement module
	dv.advert = advertModule{
		dv:       dv,
		bootTime: uint64(time.Now().Unix()),
		seq:      0,
		objDir:   object.NewMemoryFifoDir(32), // keep last few advertisements
	}

	// Create sync groups
	dv.pfxSvs = ndn_sync.NewSvSync(ndn_sync.SvSyncOpts{
		Client:      dv.client,
		GroupPrefix: config.PrefixTableSyncPrefix(),
		OnUpdate:    func(ssu ndn_sync.SvSyncUpdate) { go dv.onPfxSyncUpdate(ssu) },
		BootTime:    dv.advert.bootTime,
	})

	// Create tables
	dv.neighbors = table.NewNeighborTable(config, dv.nfdc)
	dv.rib = table.NewRib(config)
	dv.pfx = table.NewPrefixTable(config, dv.client, dv.pfxSvs)
	dv.fib = table.NewFib(config, dv.nfdc)

	return dv, nil
}

// Log identifier for the DV router.
func (dv *Router) String() string {
	return "dv-router"
}

// Start the DV router. Blocks until Stop() is called.
func (dv *Router) Start() (err error) {
	log.Info(dv, "Starting DV router", "version", utils.NDNdVersion)
	defer log.Info(dv, "Stopped DV router")

	// Initialize channels
	dv.stop = make(chan bool, 1)

	// Register neighbor faces
	dv.createFaces()
	defer dv.destroyFaces()

	// Start timers
	dv.heartbeat = time.NewTicker(dv.config.AdvertisementSyncInterval())
	dv.deadcheck = time.NewTicker(dv.config.RouterDeadInterval())
	defer dv.heartbeat.Stop()
	defer dv.deadcheck.Stop()

	// Start object client
	dv.client.Start()
	defer dv.client.Stop()

	// Start management thread
	go dv.nfdc.Start()
	defer dv.nfdc.Stop()

	// Configure face
	if err = dv.configureFace(); err != nil {
		return err
	}

	// Register interest handlers
	if err = dv.register(); err != nil {
		return err
	}

	// Start sync groups
	dv.pfxSvs.Start()
	defer dv.pfxSvs.Stop()

	// Add self to the RIB and make initial advertisement
	dv.rib.Set(dv.config.RouterName(), dv.config.RouterName(), 0)
	dv.advert.generate()

	for {
		select {
		case <-dv.heartbeat.C:
			dv.advert.sendSyncInterest()
		case <-dv.deadcheck.C:
			dv.checkDeadNeighbors()
		case <-dv.stop:
			return nil
		}
	}
}

// Stop the DV router.
func (dv *Router) Stop() {
	dv.stop <- true
}

// Configure the face to forwarder.
func (dv *Router) configureFace() (err error) {
	// Enable local fields on face. This includes incoming face indication.
	dv.nfdc.Exec(nfdc.NfdMgmtCmd{
		Module: "faces",
		Cmd:    "update",
		Args: &mgmt.ControlArgs{
			Mask:  utils.IdPtr(mgmt.FaceFlagLocalFieldsEnabled),
			Flags: utils.IdPtr(mgmt.FaceFlagLocalFieldsEnabled),
		},
		Retries: -1,
	})

	return nil
}

// Register interest handlers for DV prefixes.
func (dv *Router) register() (err error) {
	// Advertisement Sync (active)
	err = dv.engine.AttachHandler(dv.config.AdvertisementSyncActivePrefix(),
		func(args ndn.InterestHandlerArgs) {
			go dv.advert.OnSyncInterest(args, true)
		})
	if err != nil {
		return err
	}

	// Advertisement Sync (passive)
	err = dv.engine.AttachHandler(dv.config.AdvertisementSyncPassivePrefix(),
		func(args ndn.InterestHandlerArgs) {
			go dv.advert.OnSyncInterest(args, false)
		})
	if err != nil {
		return err
	}

	// Router management
	err = dv.engine.AttachHandler(dv.config.MgmtPrefix(),
		func(args ndn.InterestHandlerArgs) {
			go dv.mgmtOnInterest(args)
		})
	if err != nil {
		return err
	}

	// Register routes to forwarder
	pfxs := []enc.Name{
		dv.config.AdvertisementSyncPrefix(),
		dv.config.AdvertisementDataPrefix(),
		dv.config.PrefixTableSyncPrefix(),
		dv.config.RouterDataPrefix(),
		dv.config.MgmtPrefix(),
	}
	for _, prefix := range pfxs {
		dv.nfdc.Exec(nfdc.NfdMgmtCmd{
			Module: "rib",
			Cmd:    "register",
			Args: &mgmt.ControlArgs{
				Name:   prefix,
				Cost:   utils.IdPtr(uint64(0)),
				Origin: utils.IdPtr(config.NlsrOrigin),
			},
			Retries: -1,
		})
	}

	// Set strategy to multicast for sync prefixes
	pfxs = []enc.Name{
		dv.config.AdvertisementSyncPrefix(),
		dv.config.PrefixTableSyncPrefix(),
	}
	for _, prefix := range pfxs {
		dv.nfdc.Exec(nfdc.NfdMgmtCmd{
			Module: "strategy-choice",
			Cmd:    "set",
			Args: &mgmt.ControlArgs{
				Name: prefix,
				Strategy: &mgmt.Strategy{
					Name: config.MulticastStrategy,
				},
			},
			Retries: -1,
		})
	}

	return nil
}

// createFaces creates faces to all neighbors.
func (dv *Router) createFaces() {
	for i, neighbor := range dv.config.Neighbors {
		var mtu *uint64 = nil
		if neighbor.Mtu > 0 {
			mtu = utils.IdPtr(neighbor.Mtu)
		}

		faceId, created, err := dv.nfdc.CreateFace(&mgmt.ControlArgs{
			Uri:             utils.IdPtr(neighbor.Uri),
			FacePersistency: utils.IdPtr(uint64(mgmt.PersistencyPermanent)),
			Mtu:             mtu,
		})
		if err != nil {
			log.Error(dv, "Failed to create face to neighbor", "uri", neighbor.Uri, "err", err)
			continue
		}
		log.Info(dv, "Created face to neighbor", "uri", neighbor.Uri, "faceId", faceId)

		dv.mutex.Lock()
		dv.config.Neighbors[i].FaceId = faceId
		dv.config.Neighbors[i].Created = created
		dv.mutex.Unlock()

		dv.nfdc.Exec(nfdc.NfdMgmtCmd{
			Module: "rib",
			Cmd:    "register",
			Args: &mgmt.ControlArgs{
				Name:   dv.config.AdvertisementSyncActivePrefix(),
				Cost:   utils.IdPtr(uint64(1)),
				Origin: utils.IdPtr(config.NlsrOrigin),
				FaceId: utils.IdPtr(faceId),
			},
			Retries: 3,
		})
	}
}

// destroyFaces synchronously destroys our faces to neighbors.
func (dv *Router) destroyFaces() {
	for _, neighbor := range dv.config.Neighbors {
		if neighbor.FaceId == 0 {
			continue
		}

		dv.engine.ExecMgmtCmd("rib", "unregister", &mgmt.ControlArgs{
			Name:   dv.config.AdvertisementSyncActivePrefix(),
			Origin: utils.IdPtr(config.NlsrOrigin),
			FaceId: utils.IdPtr(neighbor.FaceId),
		})

		// only destroy faces that we created
		if neighbor.Created {
			dv.engine.ExecMgmtCmd("faces", "destroy", &mgmt.ControlArgs{
				FaceId: utils.IdPtr(neighbor.FaceId),
			})
		}
	}
}
