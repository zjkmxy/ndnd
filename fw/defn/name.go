package defn

import enc "github.com/named-data/ndnd/std/encoding"

// Localhost prefix for NFD
var LOCAL_PREFIX = enc.Name{enc.LOCALHOST, enc.NewStringComponent(enc.TypeGenericNameComponent, "nfd")}

// Non-local prefix for NFD
var NON_LOCAL_PREFIX = enc.Name{enc.LOCALHOP, enc.NewStringComponent(enc.TypeGenericNameComponent, "nfd")}

// Prefix for all stratgies
var STRATEGY_PREFIX = append(LOCAL_PREFIX, enc.NewStringComponent(enc.TypeGenericNameComponent, "strategy"))

// Default forwarding strategy name
var DEFAULT_STRATEGY = append(STRATEGY_PREFIX,
	enc.NewStringComponent(enc.TypeGenericNameComponent, "best-route"),
	enc.NewVersionComponent(1))
