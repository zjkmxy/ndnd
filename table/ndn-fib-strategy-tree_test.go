/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import (
	"sort"
	"testing"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"

	"github.com/stretchr/testify/assert"
)

func TestNdnFindNextHops(t *testing.T) {
	newFibStrategyTableTree()

	assert.NotNil(t, FibStrategyTable)

	// Root entry has no hops
	name1, _ := enc.NameFromStr("/")
	nexthops1 := FibStrategyTable.FindNextHops1(&name1)
	assert.Equal(t, 0, len(nexthops1))

	// Next hops need to be explicitly added
	name2, _ := enc.NameFromStr("/test")
	nexthops2 := FibStrategyTable.FindNextHops1(&name2)
	assert.Equal(t, 0, len(nexthops2))
	FibStrategyTable.InsertNextHop1(&name2, 25, 1)
	FibStrategyTable.InsertNextHop1(&name2, 101, 10)
	nexthops2a := FibStrategyTable.FindNextHops1(&name2)
	assert.Equal(t, 2, len(nexthops2a))
	assert.Equal(t, uint64(25), nexthops2a[0].Nexthop)
	assert.Equal(t, uint64(1), nexthops2a[0].Cost)
	assert.Equal(t, uint64(101), nexthops2a[1].Nexthop)
	assert.Equal(t, uint64(10), nexthops2a[1].Cost)

	// Check longest prefix match, should match with /test
	// and then return its next hops
	name3, _ := enc.NameFromStr("/test/name/202=abc123")
	nexthops3 := FibStrategyTable.FindNextHops1(&name3)
	assert.Equal(t, 2, len(nexthops3))
	assert.Equal(t, uint64(25), nexthops3[0].Nexthop)
	assert.Equal(t, uint64(1), nexthops3[0].Cost)
	assert.Equal(t, uint64(101), nexthops3[1].Nexthop)
	assert.Equal(t, uint64(10), nexthops3[1].Cost)
	nexthops1a := FibStrategyTable.FindNextHops1(&name1)
	assert.Equal(t, 0, len(nexthops1a))

	// Next hops should be updated when they're removed
	FibStrategyTable.RemoveNextHop1(&name2, 25)
	nexthops2b := FibStrategyTable.FindNextHops1(&name2)
	assert.Equal(t, 1, len(nexthops2b))
	assert.Equal(t, uint64(101), nexthops2b[0].Nexthop)
	assert.Equal(t, uint64(10), nexthops2b[0].Cost)

	// Test pruning
	name4, _ := enc.NameFromStr("/test4")
	name5, _ := enc.NameFromStr("/test5")
	FibStrategyTable.InsertNextHop1(&name4, 25, 1)
	FibStrategyTable.InsertNextHop1(&name5, 25, 1)

	FibStrategyTable.RemoveNextHop1(&name4, 25)
	nexthops2c := FibStrategyTable.FindNextHops1(&name4)
	assert.Equal(t, 0, len(nexthops2c))
}

func TestNdnFind_Set_Unset_Strategy(t *testing.T) {
	newFibStrategyTableTree()

	assert.NotNil(t, FibStrategyTable)

	bestRoute, _ := enc.NameFromStr("/localhost/nfd/strategy/best-route/v=1")
	multicast, _ := enc.NameFromStr("/localhost/nfd/strategy/multicast/v=1")

	name1, _ := enc.NameFromStr("/")
	assert.True(t, bestRoute.Equal(*FibStrategyTable.FindStrategy1(&name1)))

	name2, _ := enc.NameFromStr("/test")
	assert.True(t, bestRoute.Equal(*FibStrategyTable.FindStrategy1(&name2)))
	FibStrategyTable.SetStrategy1(&name2, &multicast)
	assert.True(t, bestRoute.Equal(*FibStrategyTable.FindStrategy1(&name1)))
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name2)))

	name3, _ := enc.NameFromStr("/test/name/202=abc123")
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name3)))
	FibStrategyTable.SetStrategy1(&name3, &bestRoute)
	assert.True(t, bestRoute.Equal(*FibStrategyTable.FindStrategy1(&name1)))
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name2)))
	assert.True(t, bestRoute.Equal(*FibStrategyTable.FindStrategy1(&name3)))

	// Test pruning
	FibStrategyTable.UnsetStrategy1(&name3)
	assert.True(t, bestRoute.Equal(*FibStrategyTable.FindStrategy1(&name1)))
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name2)))
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name3)))

	FibStrategyTable.SetStrategy1(&name1, &multicast)
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name1)))
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name2)))
	assert.True(t, multicast.Equal(*FibStrategyTable.FindStrategy1(&name3)))
}

func TestNdnInsertNextHop(t *testing.T) {
	newFibStrategyTableTree()
	assert.NotNil(t, FibStrategyTable)

	name, _ := enc.NameFromStr("/test/name")

	// Insert new hop
	FibStrategyTable.InsertNextHop1(&name, 100, 10)
	nextHops := FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, uint64(100), nextHops[0].Nexthop)
	assert.Equal(t, uint64(10), nextHops[0].Cost)

	// Update cost of current hop
	FibStrategyTable.InsertNextHop1(&name, 100, 20)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, uint64(100), nextHops[0].Nexthop)
	assert.NotEqual(t, uint64(10), nextHops[0].Cost)
	assert.Equal(t, uint64(20), nextHops[0].Cost)
}

func TestNdnClearNextHops(t *testing.T) {
	newFibStrategyTableTree()
	assert.NotNil(t, FibStrategyTable)

	name, _ := enc.NameFromStr("/test/name")

	// Insert new hop
	FibStrategyTable.InsertNextHop1(&name, 100, 10)
	FibStrategyTable.InsertNextHop1(&name, 100, 20)
	FibStrategyTable.InsertNextHop1(&name, 200, 10)
	FibStrategyTable.InsertNextHop1(&name, 300, 10)

	nextHops := FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 3, len(nextHops))

	FibStrategyTable.ClearNextHops1(&name)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 0, len(nextHops))

	// Should have no effect on a name with no hops
	// Or an nonexistent name
	FibStrategyTable.ClearNextHops1(&name)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 0, len(nextHops))

	nameDoesNotExist, _ := enc.NameFromStr("/asdf")
	FibStrategyTable.ClearNextHops1(&nameDoesNotExist)
	nextHops = FibStrategyTable.FindNextHops1(&nameDoesNotExist)
	assert.Equal(t, 0, len(nextHops))

	// Should only clear hops for exact match
	FibStrategyTable.InsertNextHop1(&name, 100, 10)
	nameLonger, _ := enc.NameFromStr("/test/name/longer")
	FibStrategyTable.InsertNextHop1(&nameLonger, 200, 10)

	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 1, len(nextHops))
	FibStrategyTable.ClearNextHops1(&name)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 0, len(nextHops))

	nextHops = FibStrategyTable.FindNextHops1(&nameLonger)
	assert.Equal(t, 1, len(nextHops))
}

func TestNdnRemoveNextHop(t *testing.T) {
	newFibStrategyTableTree()
	assert.NotNil(t, FibStrategyTable)

	name, _ := enc.NameFromStr("/test")

	// Insert new hop
	hopId1 := uint64(100)
	hopId2 := uint64(200)
	hopId3 := uint64(300)
	FibStrategyTable.InsertNextHop1(&name, hopId1, 10)
	FibStrategyTable.InsertNextHop1(&name, hopId2, 10)
	FibStrategyTable.InsertNextHop1(&name, hopId3, 10)
	FibStrategyTable.InsertNextHop1(&name, hopId1, 20) // updates it in place

	nextHops := FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 3, len(nextHops))

	FibStrategyTable.RemoveNextHop1(&name, hopId1)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 2, len(nextHops))

	FibStrategyTable.RemoveNextHop1(&name, hopId2)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 1, len(nextHops))

	FibStrategyTable.RemoveNextHop1(&name, hopId3)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 0, len(nextHops))

	FibStrategyTable.InsertNextHop1(&name, hopId1, 10)
	nameLonger, _ := enc.NameFromStr("/test/name/longer")
	FibStrategyTable.InsertNextHop1(&nameLonger, hopId2, 10)

	FibStrategyTable.RemoveNextHop1(&name, hopId1)
	nextHops = FibStrategyTable.FindNextHops1(&name)
	assert.Equal(t, 0, len(nextHops))
	nextHops = FibStrategyTable.FindNextHops1(&nameLonger)
	assert.Equal(t, 1, len(nextHops))
}

func TestNdnGetAllFIBEntries(t *testing.T) {
	newFibStrategyTableTree()
	assert.NotNil(t, FibStrategyTable)

	bestRoute, _ := enc.NameFromStr("/localhost/nfd/strategy/best-route/v=1")
	multicast, _ := enc.NameFromStr("/localhost/nfd/strategy/multicast/v=1")

	hopId2 := uint64(200)
	hopId3 := uint64(300)

	// Only strategy, no next hops, so it shouldn't be returned
	name, _ := enc.NameFromStr("/test")
	FibStrategyTable.SetStrategy1(&name, &multicast)

	name2, _ := enc.NameFromStr("/test/name/202=abc123")
	FibStrategyTable.SetStrategy1(&name2, &bestRoute)
	FibStrategyTable.InsertNextHop1(&name2, hopId2, 20)
	FibStrategyTable.InsertNextHop1(&name2, hopId3, 30)

	// name3 has no strategy
	name3, _ := enc.NameFromStr("/test/name_second/202=abc123")
	FibStrategyTable.InsertNextHop1(&name3, hopId3, 40)
	FibStrategyTable.InsertNextHop1(&name3, hopId3, 50)

	fse := FibStrategyTable.GetAllFIBEntries()
	assert.Equal(t, 2, len(fse))

	sort.Slice(fse, func(i, j int) bool {
		// Sort by name
		return fse[i].EncName().String() < fse[j].EncName().String()
	})

	assert.True(t, name2.Equal(*fse[0].EncName()))
	assert.True(t, bestRoute.Equal(*fse[0].GetEncStrategy()))
	nextHops := fse[0].GetNextHops()
	assert.Equal(t, 2, len(nextHops))
	assert.Equal(t, hopId2, nextHops[0].Nexthop)
	assert.Equal(t, uint64(20), nextHops[0].Cost)
	assert.Equal(t, hopId3, nextHops[1].Nexthop)
	assert.Equal(t, uint64(30), nextHops[1].Cost)

	// Only next hops, no strategy
	assert.True(t, name3.Equal(*fse[1].EncName()))
	assert.Nil(t, fse[1].GetEncStrategy())
	nextHops = fse[1].GetNextHops()
	assert.Equal(t, 1, len(nextHops))
	assert.Equal(t, hopId3, nextHops[0].Nexthop)
	assert.Equal(t, uint64(50), nextHops[0].Cost)
}

func TestNdnGetAllForwardingStrategies(t *testing.T) {
	newFibStrategyTableTree()
	assert.NotNil(t, FibStrategyTable)

	bestRoute, _ := enc.NameFromStr("/localhost/nfd/strategy/best-route/v=1")
	multicast, _ := enc.NameFromStr("/localhost/nfd/strategy/multicast/v=1")

	hopId2 := uint64(200)
	hopId3 := uint64(300)

	// No strategy, so it shouldn't be included
	name, _ := enc.NameFromStr("/test")
	FibStrategyTable.InsertNextHop1(&name, hopId2, 20)

	name2, _ := enc.NameFromStr("/test/name/202=abc123")
	FibStrategyTable.SetStrategy1(&name2, &bestRoute)
	FibStrategyTable.InsertNextHop1(&name2, hopId2, 20)
	FibStrategyTable.InsertNextHop1(&name2, hopId3, 30)

	name3, _ := enc.NameFromStr("/test/name_second/202=abc123")
	FibStrategyTable.SetStrategy1(&name3, &multicast)

	fse := FibStrategyTable.GetAllForwardingStrategies()
	// Here, the "/" has a default strategy, bestRoute in this case
	assert.Equal(t, 3, len(fse))

	sort.Slice(fse, func(i, j int) bool {
		// Sort by name
		return fse[i].EncName().String() < fse[j].EncName().String()
	})

	rootName, _ := enc.NameFromStr("/")
	assert.True(t, rootName.Equal(*fse[0].EncName()))
	assert.True(t, bestRoute.Equal(*fse[0].GetEncStrategy()))

	assert.True(t, name2.Equal(*fse[1].EncName()))
	assert.True(t, bestRoute.Equal(*fse[1].GetEncStrategy()))
	nextHops := fse[1].GetNextHops()
	assert.Equal(t, 2, len(nextHops))
	assert.Equal(t, hopId2, nextHops[0].Nexthop)
	assert.Equal(t, uint64(20), nextHops[0].Cost)
	assert.Equal(t, hopId3, nextHops[1].Nexthop)
	assert.Equal(t, uint64(30), nextHops[1].Cost)

	assert.True(t, name3.Equal(*fse[2].EncName()))
	assert.True(t, multicast.Equal(*fse[2].GetEncStrategy()))
	nextHops = fse[2].GetNextHops()
	assert.Equal(t, 0, len(nextHops))
}
