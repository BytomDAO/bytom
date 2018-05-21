package pex

import (
	"encoding/json"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"
)

type addrBookJSON struct {
	Key   string
	Addrs []*knownAddress
}

func (a *AddrBook) SaveToFile() error {
	a.mtx.RLock()
	defer a.mtx.RUnlock()

	aJSON := &addrBookJSON{Key: a.key, Addrs: []*knownAddress{}}
	for _, ka := range a.addrLookup {
		aJSON.Addrs = append(aJSON.Addrs, ka)
	}

	rawDats, err := json.MarshalIndent(aJSON, "", "\t")
	if err != nil {
		return err
	}
	return cmn.WriteFileAtomic(a.filePath, rawDats, 0644)
}

func (a *AddrBook) loadFromFile() error {
	if _, err := os.Stat(a.filePath); os.IsNotExist(err) {
		return nil
	}

	r, err := os.Open(a.filePath)
	if err != nil {
		return err
	}

	defer r.Close()
	aJSON := &addrBookJSON{}
	if err = json.NewDecoder(r).Decode(aJSON); err != nil {
		return err
	}

	a.key = aJSON.Key
	for _, ka := range aJSON.Addrs {
		a.addrLookup[ka.Addr.String()] = ka
		for _, bucketIndex := range ka.Buckets {
			bucket := a.getBucket(ka.BucketType, bucketIndex)
			bucket[ka.Addr.String()] = ka
		}
		if ka.BucketType == bucketTypeNew {
			a.nNew++
		} else {
			a.nOld++
		}
	}
	return nil
}

func (a *AddrBook) saveRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	for {
		select {
		case <-ticker.C:
			if err := a.SaveToFile(); err != nil {
				log.WithField("err", err).Error("failed to save AddrBook to file")
			}
		case <-a.Quit:
			a.SaveToFile()
			return
		}
	}
}
