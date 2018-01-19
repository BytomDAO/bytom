package aihash

import (
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

func InitData(height uint64, blockHashs []*bc.Hash) (md miningData) {
	if height < 128 {
		sha256 := makeHasher(sha3.New256())
		seed := make([]byte, 32)
		sha256(md.seed, seed)
	} else {
		md.generateSeed(blockHashs)
	}

	log.WithFields(log.Fields{
		"height": height,
		"seed":   bc.BytesToHash(md.seed).String(),
	}).Debug("Created new seed.")

	md.generateCache()

	return
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
