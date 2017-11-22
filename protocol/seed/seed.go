package seed

import (
	"github.com/bytom/consensus/algorithm"
	"github.com/bytom/protocol/bc"
	"github.com/golang/groupcache/lru"
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

	cache, err := algorithm.CreateCache(seed)
	if err != nil {
		return nil, err
	}

	s.cache.Add(seed, cache)
	return cache, nil
}
