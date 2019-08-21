package security

import "sync"

type Filter interface {
	DoFilter(string, string) error
}

type PeerFilter struct {
	filterChain []Filter
	mtx         sync.RWMutex
}

func NewPeerFilter() *PeerFilter {
	return &PeerFilter{
		filterChain: make([]Filter, 0),
	}
}

func (pf *PeerFilter) register(filter Filter) {
	pf.mtx.Lock()
	defer pf.mtx.Unlock()

	pf.filterChain = append(pf.filterChain, filter)
}

func (pf *PeerFilter) doFilter(ip string, pubKey string) error {
	pf.mtx.RLock()
	defer pf.mtx.RUnlock()

	for _, filter := range pf.filterChain {
		if err := filter.DoFilter(ip, pubKey); err != nil {
			return err
		}
	}

	return nil
}
