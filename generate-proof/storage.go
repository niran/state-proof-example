package generate_proof

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type ProofData struct {
	BlockHash   common.Hash
	BlockNumber *big.Int
	HeaderRLP   []byte
	StateRoot   common.Hash
	Proof       *gethclient.AccountResult
}

/**
 * Collect all of the data needed to verify a MPT state proof for a given slot.
 *
 * `cast proof` and `cast block` each give some of the needed data, but not the
 * full RLP-encoded block header of the specific block in which the state was proven.
 */
func GetStorageProof(r *rpc.Client, contractAddressHex string, slot int64, blockNumber *big.Int) (*ProofData, error) {
	contractAddress := common.HexToAddress(contractAddressHex)

	// Confirm that there's actually contract code at the provided address
	ec := ethclient.NewClient(r)
	code, err := ec.CodeAt(context.Background(), contractAddress, blockNumber)
	if err != nil {
		return nil, err
	}
	if len(code) == 0 {
		return nil, errors.New("no code at given address")
	}

	block, err := ec.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		return nil, err
	}

	gc := gethclient.New(r)
	storageKey := common.BigToHash(big.NewInt(slot))
	result, err := gc.GetProof(context.Background(), contractAddress, []string{storageKey.String()}, block.Number())
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	header := block.Header()
	header.EncodeRLP(&buf)

	return &ProofData{
		BlockHash:   header.Hash(),
		BlockNumber: block.Number(),
		HeaderRLP:   buf.Bytes(),
		StateRoot:   header.Root,
		Proof:       result,
	}, nil
}

func (pd *ProofData) Log() {
	log.Printf("Block number: %v", pd.BlockNumber)
	log.Printf("Block hash: %v", pd.BlockHash)
	log.Printf("State root: %v", pd.StateRoot)
	log.Printf("Header RLP: %v", common.Bytes2Hex(pd.HeaderRLP))
	text, _ := json.MarshalIndent(pd.Proof, "", "  ")
	log.Printf("Proof:\n%v", string(text))
}
