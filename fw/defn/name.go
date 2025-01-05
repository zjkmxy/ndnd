package defn

import enc "github.com/named-data/ndnd/std/encoding"

var NFD_COMP = enc.NewStringComponent(enc.TypeGenericNameComponent, "nfd")
var STRATEGY_COMP = enc.NewStringComponent(enc.TypeGenericNameComponent, "strategy")
var NLSR_COMP = enc.NewStringComponent(enc.TypeGenericNameComponent, "nlsr")

// Localhost prefix for NFD
var LOCAL_PREFIX = enc.Name{enc.LOCALHOST, NFD_COMP}

// Non-local prefix for NFD
var NON_LOCAL_PREFIX = enc.Name{enc.LOCALHOP, NFD_COMP}

// Prefix for all stratgies
var STRATEGY_PREFIX = append(LOCAL_PREFIX, STRATEGY_COMP)

// Default forwarding strategy name
var DEFAULT_STRATEGY = append(STRATEGY_PREFIX,
	enc.NewStringComponent(enc.TypeGenericNameComponent, "best-route"),
	enc.NewVersionComponent(1))
