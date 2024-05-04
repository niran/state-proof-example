package main

import (
	"context"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/prysmaticlabs/prysm/v5/api/client/beacon"

	generate_proof "github.com/niran/state-proof-example/generate-proof"
)

const BeaconNodeAPIURL = ""
const ExecutionNodeAPIURL = "https://ethereum-sepolia-rpc.publicnode.com"
const ContractAddress = "0x45b924Ee3EE404E4a9E2a3AFD0AD357eFf79fC49"
const StorageSlot = 0

func main() {
	// TODO: This script currently generates proofs against the *latest finalized*
	// beacon block, but this will be difficult to use in practice because we
	// don't know the timestamp that beacon root has been stored at on L2. Using
	// EIP 4788 on L2 requires querying the L2 for the beacon block to use. The L2
	// block header contains the current timestamp and the parent beacon root, so
	// the easiest way to sync state is to watch the L2 for new blocks, check each
	// new beacon root to see if its linked to the desired execution block (or
	// later), and then generate a proof against that *specific beacon block*.
	// Since we got the beacon block hash from the L2 alongside the L2 block
	// timestamp, we know which timestamp to submit to the EIP 4788 oracle to get
	// back the beacon root we need.

	ctx := context.Background()
	var executionBlockNumber *big.Int

	if BeaconNodeAPIURL == "" {
		log.Printf("Beacon node API URL is not set, skipping beacon proof generation")
	} else {
		// We're currently using configs.Mainnet, but that's not guaranteed to work.
		// We might need to load the "Bepolia" config YAML to generate valid proofs
		// on Sepolia.
		// https://github.com/eth-clients/sepolia/blob/main/bepolia/config.yaml
		bp, err := generate_proof.GetBeaconProof(ctx, BeaconNodeAPIURL, configs.Mainnet, beacon.IdFinalized)
		if err != nil {
			log.Fatalf("Failed to get beacon proof: %v", err)
		}
		bp.Log()
		executionBlockNumber = big.NewInt(int64(bp.BlockNumber))
	}

	r, err := rpc.Dial(ExecutionNodeAPIURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	pd, err := generate_proof.GetStorageProof(r, ContractAddress, StorageSlot, executionBlockNumber)
	if err != nil {
		log.Fatalf("Failed to get proof data: %v", err)
	}
	pd.Log()
}
