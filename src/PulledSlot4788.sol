pragma solidity ^0.8.19;

import {RLPReader} from "Solidity-RLP/RLPReader.sol";
import {MerkleTrie} from "optimism/packages/contracts-bedrock/src/libraries/trie/MerkleTrie.sol";
import {SSZ} from "eip-4788-proof/SSZ.sol";


contract PulledSlot4788 {
    using RLPReader for RLPReader.RLPItem;
    using RLPReader for RLPReader.Iterator;
    using RLPReader for bytes;

    uint256 public slotValue;
    address public immutable l1Contract;
    bytes32 public immutable slotHash;
    uint64 public lastBeaconTimestamp;

    address private constant BEACON_ROOTS_ORACLE = 0x000F3df6D732807Ef1319fB7B8bB8522d0Beac02;
    uint256 private constant STATE_ROOT_GINDEX = 12034;

    event SlotUpdated(uint256 indexed slotValue, uint256 indexed lastBeaconTimestamp, bytes32 indexed stateRootHash);

    error BeaconTimestampDecreased(uint256 lastBeaconTimestamp, uint256 beaconTimestamp);
    error BeaconRootNotFound(uint64 timestamp);
    error InvalidStateRootProof(bytes32 stateRoot, bytes32 beaconRoot, uint256 beaconTimestamp);

    constructor(address _l1Contract, bytes32 _slotHash) {
        l1Contract = _l1Contract;
        slotHash = _slotHash;
    }

    /**
     *  Update the pulled slot value with Merkle proofs from the current L1 block.
     *
     *  @param stateRoot The state root within the beacon block used for our proof.
     *  @param stateRootProof The SSZ proof for the state root within the beacon block.
     *  @param accountProof The account proof from eth_getProof.
     *  @param slotProof The storage slot proof from eth_getProof.
     *  @param beaconTimestamp The timestamp of the beacon block to retrieve via EIP 4788.
     */
    function setSlotValue(
        bytes32 stateRoot,
        bytes32[] calldata stateRootProof,
        bytes[] calldata accountProof,
        bytes[] calldata slotProof,
        uint64 beaconTimestamp) external
    {
        if (beaconTimestamp <= lastBeaconTimestamp) {
            revert BeaconTimestampDecreased(lastBeaconTimestamp, beaconTimestamp);
        }
        lastBeaconTimestamp = beaconTimestamp;
        
        bytes32 beaconRoot = getBeaconBlockRoot(beaconTimestamp);
        if (!SSZ.verifyProof(stateRootProof, beaconRoot, stateRoot, STATE_ROOT_GINDEX)) {
            revert InvalidStateRootProof(stateRoot, beaconRoot, beaconTimestamp);
        }

        // MerkleTrie.get reverts if the slot does not exist.
        bytes32 accountHash = keccak256(abi.encodePacked(l1Contract));
        bytes memory accountFields = MerkleTrie.get({
            _key: abi.encodePacked(accountHash),
            _proof: accountProof,
            _root: stateRoot
        });
        bytes32 storageRoot = bytes32(accountFields.toRlpItem().toList()[2].toUint());

        slotValue = MerkleTrie.get({
            _key: abi.encodePacked(slotHash),
            _proof: slotProof,
            _root: storageRoot
        }).toRlpItem().toUint();

        emit SlotUpdated(slotValue, beaconTimestamp, stateRoot);
    }

    function getBeaconBlockRoot(uint64 ts)
        internal
        view
        returns (bytes32 blockRoot)
    {
        (bool success, bytes memory data) =
            BEACON_ROOTS_ORACLE.staticcall(abi.encode(ts));

        if (!success || data.length == 0) {
            revert BeaconRootNotFound(ts);
        }

        blockRoot = abi.decode(data, (bytes32));
    }
}
