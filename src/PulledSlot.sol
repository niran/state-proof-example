pragma solidity ^0.8.19;

import {RLPReader} from "Solidity-RLP/RLPReader.sol";
import {MerkleTrie} from "optimism/packages/contracts-bedrock/src/libraries/trie/MerkleTrie.sol";

interface IL1Block {
    /// @notice The latest L1 blockhash.
    function hash() external returns (bytes32);
}

contract PulledSlot {
    using RLPReader for RLPReader.RLPItem;
    using RLPReader for RLPReader.Iterator;
    using RLPReader for bytes;

    uint256 public slotValue;
    address public immutable l1Contract;
    bytes32 public immutable slotHash;
    IL1Block public immutable l1Block;

    uint256 constant HEADER_STATE_ROOT_INDEX = 3;
    uint256 constant HEADER_NUMBER_INDEX = 8;
    uint256 constant HEADER_TIMESTAMP_INDEX = 11;

    struct BlockHeader {
        bytes32 hash;
        bytes32 stateRootHash;
        uint256 number;
        uint256 timestamp;
    }

    error BlockHashMismatch(bytes32 expected, bytes32 actual);

    event SlotUpdated(uint256 indexed slotValue, uint256 indexed l1BlockNumber, bytes32 indexed l1BlockHash);

    constructor(address _l1Contract, bytes32 _slotHash, address _l1Block) {
        l1Contract = _l1Contract;
        slotHash = _slotHash;
        l1Block = IL1Block(_l1Block);
    }

    /**
     *  Update the pulled slot value with Merkle proofs from the current L1 block.
     *
     *  @param blockHeaderRlp The RLP-encoded L1 block header from eth_getBlockByNumber.
     *  @param accountProof The account proof from eth_getProof.
     *  @param slotProof The storage slot proof from eth_getProof.
     */
    function setSlotValue(
        bytes memory blockHeaderRlp,
        bytes[] memory accountProof,
        bytes[] memory slotProof) external
    {
        BlockHeader memory header = parseBlockHeader(blockHeaderRlp);
        if (header.hash != l1Block.hash()) {
            revert BlockHashMismatch(l1Block.hash(), header.hash);
        }

        // MerkleTrie.get reverts if the slot does not exist.
        bytes32 accountHash = keccak256(abi.encodePacked(l1Contract));
        bytes memory accountFields = MerkleTrie.get({
            _key: abi.encodePacked(accountHash),
            _proof: accountProof,
            _root: header.stateRootHash
        });
        bytes32 storageRoot = bytes32(accountFields.toRlpItem().toList()[2].toUint());

        slotValue = MerkleTrie.get({
            _key: abi.encodePacked(slotHash),
            _proof: slotProof,
            _root: storageRoot
        }).toRlpItem().toUint();

        emit SlotUpdated(slotValue, header.number, header.hash);
    }

    /**
     * @notice Parses RLP-encoded block header.
     * @param _headerRlpBytes RLP-encoded block header.
     *
     * Extracted from curve-merkle-oracle.
     * https://github.com/lidofinance/curve-merkle-oracle/blob/fffd375659358af54a6e8bbf8c3aa44188894c81/contracts/StateProofVerifier.sol
     */
    function parseBlockHeader(bytes memory _headerRlpBytes)
        internal pure returns (BlockHeader memory)
    {
        BlockHeader memory result;
        RLPReader.RLPItem[] memory headerFields = _headerRlpBytes.toRlpItem().toList();

        require(headerFields.length > HEADER_TIMESTAMP_INDEX);

        result.stateRootHash = bytes32(headerFields[HEADER_STATE_ROOT_INDEX].toUint());
        result.number = headerFields[HEADER_NUMBER_INDEX].toUint();
        result.timestamp = headerFields[HEADER_TIMESTAMP_INDEX].toUint();
        result.hash = keccak256(_headerRlpBytes);

        return result;
    }
}
