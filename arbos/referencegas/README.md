# xgas EIP-8429 prototype

This package is a deterministic reference implementation of the current two-ratchet draft's pricing and xgas's selected 50/50 native `$xMoney` distribution. It is **not enabled on chain**. The fork is pinned to Nitro `v3.11.4` (`7d5ac271b400f710f6267ad759c6afc3e12d7059`), the version documented for xgas chain 466302.

## Confirmed xgas topology

- xgas 466302 is an Orbit **rollup** on Robinhood Chain 4663. It posts calldata to the parent and has permissionless validation. The older 466301 AnyTrust chain is retired.
- `$xMoney` is the native L4 gas token. Its USDG-backed ERC-20 vault lives on the parent chain. A native L4 balance has no contract address that can self-enroll. EIP-8429 can price calls into opted-in **L4 contracts**, including a newly deployed L4 wrapper, but it cannot directly enroll the parent vault or native-token transfers.
- The parent `$xMoney` ERC-20 already taxes transfers and burn-to-dead increases reserve NAV. The proposed reference surcharge is separate from that tax and does not change the parent vault contract.

## Package behavior

- `Engine.BeginBlock` resets fast and own counts on **L4 block sequence**, and resets slow counts only at a window boundary.
- `Engine.Reference` advances all counters before gas is deducted. Callers must invoke it only for enrolled, nonempty-code targets of top-level transactions, `CALL`, and `CALLCODE`; not for `STATICCALL` or `DELEGATECALL`.
- The fast charge requires both global ordinal greater than two and the origin's third own reference. The slow charge starts at the seventeenth reference in the window. The reference pays the larger charge.
- `Ledger` retains charge gas per enrolled account across frame reverts. `Settle` divides the native fee 50/50 between a permanent sink and application beneficiaries, with all rounding dust going to the sink.
- `solidity/src/ReferenceRegistryXgas.sol` implements the permissionless self-enrollment and delayed-exit registry against `ArbSys.arbBlockNumber()`. It is tested in its own small Foundry project. It is not installed at a reserved address yet.

## Consensus work required before activation

1. Select a fork L4 block and freeze chain constants after measuring actual L4 block cadence. A week is `W_SLOW` **L4 blocks**, not Robinhood blocks or the EVM `NUMBER` opcode.
2. Implement the registry at a reserved address with self-enrollment, a block-start snapshot, and delayed exits. The registry's window math must use `ArbSys.arbBlockNumber()` on L4.
3. Hook top-level calls and `CALL`/`CALLCODE` in the Nitro Geth fork. Charge before EIP-150's 63/64 forwarding calculation. Charge and count failed/reverted calls; don't count `STATICCALL` or `DELEGATECALL`.
4. Make the slow counter recoverable across node restarts, state sync, assertion replay, reorgs, and speculative block-building rollbacks. Keeping it only in process memory is insufficient. An ArbOS state commitment would be an xgas-specific deviation from the Ethereum draft's off-state map and must be documented.
5. Route charged `$xMoney` from the ordinary Nitro network/infra fee accounts into a permanent sink and immutable application vaults without double-minting or touching batch-poster reimbursement. Ensure fee refunds, retryables, and multi-gas discounts cannot reclaim the surcharge. Define a vault that cannot immediately pay the enroller back, or self-farming defeats the mechanism.
6. Build native and WASM binaries from identical source, set the corresponding rollup WASM module root on Robinhood, and have sequencer and validators upgrade together. A normal `scheduleArbOSUpgrade` cannot activate a version absent from stock Nitro.
7. Before mainnet activation, replay real xgas blocks on the fork and compare old/new receipts, fee accounting, balances, retryables, withdrawals, and assertion roots. Publish the fork source and exact image/WASM hashes so permissionless validators can follow.

Do not put this prototype image on the live sequencer: the execution hooks, registry, durable counter state, and fee accounting listed above are still absent.
