state-proof-example
===================

Example code for proving the state of an L1 contract's storage slots on L2.

There are two types of proofs that we handle:

* Merkle proofs from the L1 **block hash**
* Merkle proofs from the L1 **beacon root**, provided by EIP-4788

Both kinds of proofs can be tricky to work with.

Block hash proofs are the most widely supported across rollups and alt-L1 chains, but they need to be submitted to the L2 chain during the short period when it matches the L2's value for the L1 block hash. Alternatively, you could store the latest L1 block hash in your own contract on L2, then prove against that block at your own leisure.

Beacon root proofs can rely on the buffer of the most recent 8191 beacon roots (about one day), so you don't have to store your own roots or race to get your proof in. However, beacon root proofs are only usable on L2 chains that support EIP-4788, like OP Stack rollups. You also need a beacon node to generate the proofs, and public beacon nodes are hard to find.

# Generating Proofs

Configure your proof in `generate-proof/main/main.go` by setting these constants:

```
const BeaconNodeAPIURL = ""
const ExecutionNodeAPIURL = "https://ethereum-sepolia-rpc.publicnode.com"
const ContractAddress = "0x45b924Ee3EE404E4a9E2a3AFD0AD357eFf79fC49"
const StorageSlot = 0
```

## Block Hash Proofs

`go run ./generate-proof/main` generates all the values you need to pass into `PulledSlot.setSlotValue`. See `test/PulledSlot.t.sol` for an example of how to use these values.

## Beacon Root Proofs

Make sure you've set `BeaconNodeAPIURL` in `generate-proof/main/main.go`, then `go run ./generate-proof/main`. The timestamp provided by the script is unlikely to be (and almost certainly isn't) the beacon timestamp needed to verify the proof on an L2. The code to do this correctly has not been written yet.

The general workflow you will need is to wait for your state to be written on L1, get the beacon root of that block, then poll each L2 block for new timestamps to query the beacon root oracle with. Once the beacon root matches the block you're looking for (or a later block), use that beacon root as the identifier for the block to generate a proof from.

# Setup

For EIP4788 beacon root proofs to work in Foundry, you must set `evm_version = "shanghai"` in your foundry.toml file.

# Tests

```
FORK_URL=https://rpc.sepolia.org forge test
```
