package referencegas

import (
	"math/big"
	"testing"
)

func TestSettlementConservesNativeGasTokenAndRoundsToSink(t *testing.T) {
	l := NewLedger()
	l.Add(addr(1), Charge{Gas: 1})
	l.Add(addr(2), Charge{Gas: 2})
	d := l.Settle(big.NewInt(3), func(a Address) Address { return a })
	if d.Total.Cmp(big.NewInt(9)) != 0 || d.Sink.Cmp(big.NewInt(5)) != 0 ||
		d.Beneficiaries[addr(1)].Cmp(big.NewInt(1)) != 0 || d.Beneficiaries[addr(2)].Cmp(big.NewInt(3)) != 0 {
		t.Fatalf("unexpected distribution: %+v", d)
	}
}
