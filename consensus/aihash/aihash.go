package aihash

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

const (
	matSize     = 1 << 8 // Size of matrix
	matNum      = 1 << 8 // Number of matrix
	epochLength = 1 << 7 // Blocks per epoch
	mulRounds   = 64     // Number of rounds in mulmatrix
)

// CreateSeed return epoch seed, type is *bc.Hash
func CreateSeed(height uint64, blockHashs []*bc.Hash) *bc.Hash {
	if height < 128 {
		sha256 := makeHasher(sha3.New256())
		seed := make([]byte, 32)
		sha256(seed, seed)
		seedHash := bc.BytesToHash(seed)
	} else {
		seed := createSeed(blockHashs)
		seedHash := bc.BytesToHash(seed)
	}

	log.WithFields(log.Fields{
		"height": height,
		"seed":   seedHash.String(),
	}).Debug("Created new seed.")

	return &seedHash
}

// CreateCache return cache, type is []int32
func CreateCache(seed []byte) ([]uint32, error) {
	log.Debug("Start creating cache...")
	if seed == nil {
		return nil, errors.New("Seed is invalid or not exist!")
	}

	cache := generateCache(seed)

	return cache, nil
}

// AIHash verify header is correct.
func AIHash(height uint64, header *bc.Hash, cache []uint32) (*bc.Hash, error) {
	if header == nil {
		return nil, errors.New("BlockHeader Hash is invalid or not exist!")
	}

	// Bytes of cache production
	cacheLength := matSize * matSize * matNum
	// convert []byte to []uint32, so length/4
	if len(cache) != cacheLength/4 {
		return nil, errors.New("Cache is invalid!")
	}

	matList := make([]matrix.Matrix, matNum)

	fillMatrixList(matList, matSize, matNum, epochLength, cache, height)

	m := mulMatrix(matList, matSize, matNum, mulRounds, header.Bytes())
	h := hashMatrix(m, matSize)

	return h, nil
}
