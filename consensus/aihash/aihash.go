package aihash

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
)

func InitMiningData(hash128 [128]*bc.Hash) *MiningData {
	var blockHashs []*bc.Hash = hash128[:]

	var md MiningData
	md.generateSeed(blockHashs)

	log.Printf("Created new seed:%v", hex.EncodeToString(md.seed))

	md.generateCache()

	return &md
}

func AIHash(header *bc.Hash, cache []uint32) (*bc.Hash, error) {
	if header == nil {
		return nil, errors.New("BlockHeader Hash is invalid or not exist!")
	}

	if cache == nil {
		return nil, errors.New("Mining data is invalid: cache is nil!")
	}

	result := mulMatrix(cache, header.Bytes())

	hash := hashMatrix(result)

	return hash, nil
}
