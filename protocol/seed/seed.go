package seed

import (
	"github.com/golang/groupcache/lru"

	"github.com/bytom/consensus/aihash"
	"github.com/bytom/protocol/bc"
)

const maxCached = 2

type SeedCaches struct {
	cache *lru.Cache
}

func NewSeedCaches() *SeedCaches {
	return &SeedCaches{
		cache: lru.New(maxCached),
	}
}

func (s *SeedCaches) Get(seed *bc.Hash) ([]uint32, error) {
	if v, ok := s.cache.Get(seed); ok {
		return v.([]uint32), nil
	}

	md := aihash.Md

	s.cache.Add(seed, md.GetCache())
	return md.GetCache(), nil
}
