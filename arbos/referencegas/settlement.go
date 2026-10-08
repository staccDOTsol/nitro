package referencegas

import "math/big"

// Ledger accumulates surcharge gas by enrolled target for a single transaction.
// A reverted frame does not remove its charge. A rejected transaction does not
// settle and must not be added to a canonical block's counter checkpoint.
type Ledger struct {
	gas map[Address]uint64
}

func NewLedger() *Ledger { return &Ledger{gas: make(map[Address]uint64)} }

func (l *Ledger) Add(target Address, charge Charge) {
	if charge.Gas == 0 {
		return
	}
	if ^uint64(0)-l.gas[target] < charge.Gas {
		panic("reference gas ledger overflow")
	}
	l.gas[target] += charge.Gas
}

type Distribution struct {
	Total         *big.Int
	Sink          *big.Int
	Beneficiaries map[Address]*big.Int
}

// Settle splits the paid native-gas-token surcharge 50/50. Individual
// beneficiary shares use integer division; every rounding remainder goes to
// the sink. The beneficiary lookup must use the block-start registry snapshot.
func (l *Ledger) Settle(effectiveGasPrice *big.Int, beneficiaryOf func(Address) Address) Distribution {
	if effectiveGasPrice == nil || effectiveGasPrice.Sign() < 0 {
		panic("invalid effective gas price")
	}
	totalGas := new(big.Int)
	for _, gas := range l.gas {
		totalGas.Add(totalGas, new(big.Int).SetUint64(gas))
	}
	total := new(big.Int).Mul(totalGas, effectiveGasPrice)
	beneficiaryBudget := new(big.Int).Sub(total, new(big.Int).Div(new(big.Int).Set(total), big.NewInt(2)))
	result := Distribution{Total: total, Sink: new(big.Int).Set(total), Beneficiaries: make(map[Address]*big.Int)}
	if totalGas.Sign() == 0 {
		return result
	}
	for target, gas := range l.gas {
		share := new(big.Int).Mul(beneficiaryBudget, new(big.Int).SetUint64(gas))
		share.Div(share, totalGas)
		beneficiary := beneficiaryOf(target)
		if result.Beneficiaries[beneficiary] == nil {
			result.Beneficiaries[beneficiary] = new(big.Int)
		}
		result.Beneficiaries[beneficiary].Add(result.Beneficiaries[beneficiary], share)
		result.Sink.Sub(result.Sink, share)
	}
	return result
}
