// Package referencegas implements the deterministic counters and pricing rule
// from the EIP-8429 draft. It deliberately contains no fee transfer or EVM hook:
// those must be integrated with Nitro's consensus execution and ArbOS accounting.
package referencegas

import "math"

type Address [20]byte

type Key struct {
	Target Address
	Origin Address
}

type Policy struct {
	WindowBlocks uint64
	FreeBlock    uint64
	OwnMinimum   uint64
	FreeWindow   uint64
	FastBase     uint64
	FastCap      uint64
	SlowBase     uint64
	SlowCap      uint64
}

// OrbitDraft uses the draft's recommended fast base and week-sized window for
// a chain producing one block every 250 ms. These values are illustrative until
// xgas block timing and replay measurements are used to set chain constants.
var OrbitDraft = Policy{
	WindowBlocks: 2_419_200,
	FreeBlock:    2,
	OwnMinimum:   3,
	FreeWindow:   16,
	FastBase:     20_000,
	FastCap:      30_000_000,
	SlowBase:     400,
	SlowCap:      5_000_000,
}

type Charge struct {
	BlockOrdinal  uint64
	OwnOrdinal    uint64
	WindowOrdinal uint64
	FastGas       uint64
	SlowGas       uint64
	Gas           uint64
}

// Engine must be owned by block execution, never by an individual transaction.
// Its window state must be checkpointed or replayed on restart and reorg.
type Engine struct {
	Policy     Policy
	block      uint64
	window     uint64
	started    bool
	refs       map[Address]uint64
	own        map[Key]uint64
	windowRefs map[Key]uint64
}

func New(policy Policy) *Engine {
	if policy.WindowBlocks == 0 || policy.OwnMinimum == 0 || policy.FastCap == 0 || policy.SlowCap == 0 {
		panic("invalid reference gas policy")
	}
	return &Engine{Policy: policy, refs: make(map[Address]uint64), own: make(map[Key]uint64), windowRefs: make(map[Key]uint64)}
}

// BeginBlock uses the Orbit chain's own block sequence, not the EVM NUMBER
// opcode (which can report the parent chain's height on Arbitrum-family chains).
func (e *Engine) BeginBlock(number uint64) {
	if e.started && number <= e.block {
		panic("reference gas blocks must be strictly increasing; restore a checkpoint on reorg")
	}
	w := number / e.Policy.WindowBlocks
	if !e.started || w != e.window {
		e.windowRefs = make(map[Key]uint64)
	}
	e.refs = make(map[Address]uint64)
	e.own = make(map[Key]uint64)
	e.block, e.window, e.started = number, w, true
}

// Reference is called once for each counted state-changing message call. The
// caller must first check the enrollment snapshot and nonempty target code.
// It must call this before gas deduction, including for calls that run out of
// gas or revert. STATICCALL and DELEGATECALL are never passed to this method.
func (e *Engine) Reference(target, origin Address) Charge {
	if !e.started {
		panic("BeginBlock required")
	}
	key := Key{Target: target, Origin: origin}
	e.refs[target] = increment(e.refs[target])
	e.own[key] = increment(e.own[key])
	e.windowRefs[key] = increment(e.windowRefs[key])
	c := Charge{BlockOrdinal: e.refs[target], OwnOrdinal: e.own[key], WindowOrdinal: e.windowRefs[key]}
	if c.BlockOrdinal > e.Policy.FreeBlock && c.OwnOrdinal >= e.Policy.OwnMinimum {
		c.FastGas = cappedSquare(e.Policy.FastBase, c.BlockOrdinal, e.Policy.FastCap)
	}
	if c.WindowOrdinal > e.Policy.FreeWindow {
		c.SlowGas = cappedSquare(e.Policy.SlowBase, c.WindowOrdinal, e.Policy.SlowCap)
	}
	c.Gas = max(c.FastGas, c.SlowGas)
	return c
}

// Clone is a consensus-safe checkpoint for speculative block building or a
// reorg. Rollback must restore a prior clone, not decrement reverted calls.
func (e *Engine) Clone() *Engine {
	c := *e
	c.refs = cloneMap(e.refs)
	c.own = cloneMap(e.own)
	c.windowRefs = cloneMap(e.windowRefs)
	return &c
}

func cloneMap[K comparable](m map[K]uint64) map[K]uint64 {
	c := make(map[K]uint64, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func increment(v uint64) uint64 {
	if v == math.MaxUint64 {
		return v
	}
	return v + 1
}

func cappedSquare(base, ordinal, cap uint64) uint64 {
	if base == 0 {
		return 0
	}
	if ordinal > math.MaxUint64/ordinal || base > math.MaxUint64/(ordinal*ordinal) {
		return cap
	}
	v := base * ordinal * ordinal
	if v > cap {
		return cap
	}
	return v
}
