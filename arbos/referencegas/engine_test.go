package referencegas

import "testing"

func addr(v byte) Address { var a Address; a[19] = v; return a }

func TestOwnGateProtectsBystanderInBusyBlock(t *testing.T) {
	e := New(OrbitDraft)
	e.BeginBlock(100)
	target := addr(1)
	for i := byte(1); i <= 20; i++ {
		if got := e.Reference(target, addr(i)).Gas; got != 0 {
			t.Fatalf("bystander %d charged %d", i, got)
		}
	}
	origin := addr(21)
	e.Reference(target, origin)
	e.Reference(target, origin)
	got := e.Reference(target, origin)
	if got.BlockOrdinal != 23 || got.OwnOrdinal != 3 || got.Gas != 20_000*23*23 {
		t.Fatalf("unexpected third reference: %+v", got)
	}
}

func TestWindowPersistsAcrossBlocksAndResetsAtBoundary(t *testing.T) {
	p := OrbitDraft
	p.WindowBlocks = 2
	p.FreeWindow = 2
	e := New(p)
	target, origin := addr(1), addr(2)
	e.BeginBlock(0)
	e.Reference(target, origin)
	e.BeginBlock(1)
	e.Reference(target, origin)
	got := e.Reference(target, origin)
	if got.FastGas != 0 || got.SlowGas != 3_600 || got.Gas != 3_600 {
		t.Fatalf("unexpected slow charge: %+v", got)
	}
	e.BeginBlock(2)
	if got := e.Reference(target, origin); got.Gas != 0 || got.WindowOrdinal != 1 {
		t.Fatalf("window did not reset: %+v", got)
	}
}

func TestCounterSurvivesRevertedCallAndCheckpointRestoresFork(t *testing.T) {
	e := New(OrbitDraft)
	e.BeginBlock(7)
	target, origin := addr(1), addr(2)
	e.Reference(target, origin)
	checkpoint := e.Clone()
	e.Reference(target, origin) // This call might revert; it still counts.
	if got := e.Reference(target, origin); got.OwnOrdinal != 3 || got.Gas == 0 {
		t.Fatalf("reverted call was lost: %+v", got)
	}
	if got := checkpoint.Reference(target, origin); got.OwnOrdinal != 2 || got.Gas != 0 {
		t.Fatalf("checkpoint was mutated: %+v", got)
	}
}

func TestCappedSquareCannotOverflow(t *testing.T) {
	if got := cappedSquare(20_000, ^uint64(0), 30_000_000); got != 30_000_000 {
		t.Fatalf("overflow produced %d", got)
	}
}
