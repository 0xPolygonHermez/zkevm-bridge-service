package bridgetree

import (
	"golang.org/x/crypto/sha3"
)

func hash(data ...[KeyLen]byte) [KeyLen]byte {
	var res [KeyLen]byte
	hash := sha3.NewLegacyKeccak256()
	for _, d := range data {
		hash.Write(d[:]) //nolint:errcheck,gosec
	}
	copy(res[:], hash.Sum(nil))
	return res
}

// HashZero is an empty hash
var HashZero = [32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func generateZeroHashes(height uint8) [][KeyLen]byte {
	var zeroHashes = [][KeyLen]byte{
		HashZero,
	}
	for i := 1; i <= int(height); i++ {
		zeroHashes = append(zeroHashes, hash(zeroHashes[i-1], zeroHashes[i-1]))
	}
	return zeroHashes
}
