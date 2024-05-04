package generate_proof

import (
	"bytes"
	"context"
	"log"

	"github.com/ethereum/go-ethereum/common"
	zrnt_common "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
	"github.com/protolambda/ztyp/view"
	"github.com/prysmaticlabs/prysm/v5/api/client/beacon"
	"github.com/prysmaticlabs/prysm/v5/container/trie"
)

const BeaconBlockBodyPosition = 4
const ExecutionPayloadPosition = 9
const ExecutionStateRootPosition = 2

type BeaconProof struct {
	Root        []byte
	Item        []byte
	Index       uint64
	Proof       [][]byte
	BlockNumber uint64
	Timestamp   uint64
}

func GetBeaconProof(ctx context.Context, host string, bcfg *zrnt_common.Spec, blockId beacon.StateOrBlockId) (*BeaconProof, error) {
	// Use Prysm's beacon chain client to fetch the block and state from the beacon chain API.
	c, err := beacon.NewClient(host)
	if err != nil {
		return nil, err
	}

	bb, err := c.GetBlock(ctx, blockId)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(bb)
	sbbType := deneb.SignedBeaconBlockType(bcfg)
	sbb, err := view.AsContainer(sbbType.Deserialize(codec.NewDecodingReader(r, uint64(len(bb)))))
	if err != nil {
		return nil, err
	}
	blockView, err := view.AsContainer(sbb.Get(0))
	if err != nil {
		return nil, err
	}
	bodyView, err := view.AsContainer(blockView.Get(BeaconBlockBodyPosition))
	if err != nil {
		return nil, err
	}
	ep, err := view.AsContainer(bodyView.Get(ExecutionPayloadPosition))
	if err != nil {
		return nil, err
	}
	esrView, err := ep.Get(ExecutionStateRootPosition)
	if err != nil {
		return nil, err
	}
	esr := esrView.Backing().MerkleRoot(tree.Hash)

	bodyGindex, err := tree.ToGindex64(BeaconBlockBodyPosition, tree.CoverDepth(deneb.BeaconBlockType(bcfg).FieldCount()))
	if err != nil {
		return nil, err
	}

	epDepth := tree.CoverDepth(deneb.BeaconBlockBodyType(bcfg).FieldCount())
	esrDepth := tree.CoverDepth(deneb.ExecutionPayloadType(bcfg).FieldCount())
	gindex := bodyGindex
	gindex <<= epDepth
	gindex |= ExecutionPayloadPosition
	gindex <<= esrDepth
	gindex |= ExecutionStateRootPosition

	iter, _ := gindex.BitIter()
	proof := make([][]byte, 0)
	var node tree.Node = blockView.BackingNode
	for {
		nodeRoot := node.MerkleRoot(tree.Hash)
		right, ok := iter.Next()
		if !ok {
			if nodeRoot != esr {
				log.Fatalf("Expected leaf %v, got %v", common.Bytes2Hex(esr[:]), common.Bytes2Hex(nodeRoot[:]))
			}
			break
		}
		var complement tree.Node
		if right {
			complement, err = node.Left()
			if err != nil {
				return nil, err
			}
			node, err = node.Right()

		} else {
			complement, err = node.Right()
			if err != nil {
				return nil, err
			}
			node, err = node.Left()
		}
		if err != nil {
			return nil, err
		}

		// Prepend the complement to the proof since we're traversing from the root down to the leaf.
		complementRoot := complement.MerkleRoot(tree.Hash)
		proof = append([][]byte{complementRoot[:]}, proof...)
	}

	bn, err := view.AsUint64(ep.Get(6))
	if err != nil {
		return nil, err
	}
	ts, err := zrnt_common.AsTimestamp(ep.Get(9))
	if err != nil {
		return nil, err
	}
	root := blockView.HashTreeRoot(tree.Hash)

	return &BeaconProof{
		Root:        root[:],
		Item:        esr[:],
		Index:       uint64(gindex),
		Proof:       proof,
		BlockNumber: uint64(bn),
		Timestamp:   uint64(ts),
	}, nil
}

func (bp *BeaconProof) Log() {
	hexProof := make([]string, 0)
	for _, branch := range bp.Proof {
		hexProof = append(hexProof, common.Bytes2Hex(branch))
	}

	log.Printf("Beacon root: %v", common.Bytes2Hex(bp.Root))
	log.Printf("Beacon timestamp: %v", bp.Timestamp)
	log.Printf("Execution state root: %v", common.Bytes2Hex(bp.Item))
	log.Printf("Proof index: %v", bp.Index)
	log.Printf("SSZ Beacon proof: %v", hexProof)
	isValid := trie.VerifyMerkleProof(bp.Root, bp.Item, bp.Index, bp.Proof)
	log.Printf("SSZ Beacon proof is valid: %v", isValid)
}
