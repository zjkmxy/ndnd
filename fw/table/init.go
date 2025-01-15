/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"sync/atomic"
	"time"

	"github.com/named-data/ndnd/fw/core"
	enc "github.com/named-data/ndnd/std/encoding"
)

// deadNonceListLifetime is the lifetime of entries in the dead nonce list.
var deadNonceListLifetime time.Duration

// csCapacity contains the default capacity of each forwarding thread's Content Store.
var CsCapacity atomic.Int32

// csAdmit determines whether contents will be admitted to the Content Store.
var CsAdmit atomic.Bool

// csServe determines whether contents will be served from the Content Store.
var CsServe atomic.Bool

// csReplacementPolicy contains the replacement policy used by Content Stores in the forwarder.
var csReplacementPolicy string

// producerRegions contains the prefixes produced in this forwarder's region.
var producerRegions []string

// Configure configures the forwarding system.
func Configure() {
	// Content Store (mutable config)
	CsCapacity.Store(int32(core.C.Tables.ContentStore.Capacity))
	CsAdmit.Store(core.C.Tables.ContentStore.Admit)
	CsServe.Store(core.C.Tables.ContentStore.Serve)
	csReplacementPolicy = core.C.Tables.ContentStore.ReplacementPolicy

	// Dead Nonce List
	deadNonceListLifetime = time.Duration(core.C.Tables.DeadNonceList.Lifetime) * time.Millisecond

	// Network Region Table
	producerRegions = core.C.Tables.NetworkRegion.Regions
	if producerRegions == nil {
		producerRegions = make([]string, 0)
	}
	for _, region := range producerRegions {
		name, err := enc.NameFromStr(region)
		if err != nil {
			core.Log.Fatal(nil, "Could not add producer region", "name", region, "err", err)
		}
		NetworkRegion.Add(name)
		core.Log.Debug(nil, "Added producer region", "name", region)
	}
}

func CreateFIBTable() {
	switch core.C.Tables.Fib.Algorithm {
	case "hashtable":
		newFibStrategyTableHashTable(core.C.Tables.Fib.Hashtable.M)
	case "nametree":
		newFibStrategyTableTree()
	default:
		core.Log.Fatal(nil, "Unknown FIB table algorithm", "algo", core.C.Tables.Fib.Algorithm)
	}
}
