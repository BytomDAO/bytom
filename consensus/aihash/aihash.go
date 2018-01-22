package aihash

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	// "github.com/bytom/protocol/bc/legacy"
)

func InitMiningData(height uint64, blockHashs []*bc.Hash) (md miningData) {
	if height < 128 {
		sha256 := makeHasher(sha3.New256())
		seed := make([]byte, 32)
		sha256(md.seed, seed)
	} else {
		md.generateSeed(blockHashs)
	}

	log.WithFields(log.Fields{
		"height": height,
		"seed":   hex.EncodeToString(md.seed),
	}).Debug("Created new seed.")

	md.generateCache()

	return
}

//
func AIHash(header *bc.Hash, cache []uint32, md miningData) (*bc.Hash, error) {
	if header == nil {
		return nil, errors.New("BlockHeader Hash is invalid or not exist!")
	}

	if md.cache == nil {
		return nil, errors.New("Mining data is invalid: cache is nil!")
	}

	result := mulMatrix(md.cache, header.Bytes())

	hash := hashMatrix(result)

	return hash, nil
}
