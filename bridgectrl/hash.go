package bridgectrl

import (
	"encoding/binary"

	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman"
	"github.com/iden3/go-iden3-crypto/keccak256"
	"golang.org/x/crypto/sha3"
)

// Hash calculates  the keccak hash of elements.
func Hash(data ...[KeyLen]byte) [KeyLen]byte {
	var res [KeyLen]byte
	hash := sha3.NewLegacyKeccak256()
	for _, d := range data {
		hash.Write(d[:]) //nolint:errcheck,gosec
	}
	copy(res[:], hash.Sum(nil))
	return res
}

// HashZero is an empty hash
var HashZero = [KeyLen]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func generateZeroHashes(height uint8) [][KeyLen]byte {
	var zeroHashes = [][KeyLen]byte{
		HashZero,
	}
	// This generates a leaf = HashZero in position 0. In the rest of the positions that are equivalent to the ascending levels,
	// we set the hashes of the nodes. So all nodes from level i=5 will have the same value and same children nodes.
	for i := 1; i <= int(height); i++ {
		zeroHashes = append(zeroHashes, Hash(zeroHashes[i-1], zeroHashes[i-1]))
	}
	return zeroHashes
}

func hashDeposit(deposit *etherman.Deposit) [KeyLen]byte {
	var res [KeyLen]byte
	origNet := make([]byte, 4) //nolint:gomnd
	binary.BigEndian.PutUint32(origNet, uint32(deposit.OriginalNetwork))
	destNet := make([]byte, 4) //nolint:gomnd
	binary.BigEndian.PutUint32(destNet, uint32(deposit.DestinationNetwork))
	var buf [KeyLen]byte
	metaHash := keccak256.Hash(deposit.Metadata)
	copy(res[:], keccak256.Hash([]byte{deposit.LeafType}, origNet, deposit.OriginalAddress[:], destNet, deposit.DestinationAddress[:], deposit.Amount.FillBytes(buf[:]), metaHash))
	return res
}
