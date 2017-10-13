// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package pseudohsm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bytom/common"
	_ "github.com/bytom/errors"
)

// Minimum amount of time between cache reloads. This limit applies if the platform does
// not support change notifications. It also applies if the keystore directory does not
// exist yet, the code will attempt to create a watcher at most this often.
const minReloadInterval = 2 * time.Second

type keysByFile []XPub

func (s keysByFile) Len() int           { return len(s) }
func (s keysByFile) Less(i, j int) bool { return s[i].File < s[j].File }
func (s keysByFile) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// AmbiguousAddrError is returned when attempting to unlock
// an address for which more than one file exists.
type AmbiguousAddrError struct {
	Addr    common.Address
	Matches []XPub
}

func (err *AmbiguousAddrError) Error() string {
	files := ""
	for i, a := range err.Matches {
		files += a.File
		if i < len(err.Matches)-1 {
			files += ", "
		}
	}
	return fmt.Sprintf("multiple keys match address (%s)", files)
}

// addrCache is a live index of all keys in the keystore.
type addrCache struct {
	keydir   string
	watcher  *watcher
	mu       sync.Mutex
	all      keysByFile
	byAddr   map[common.Address][]XPub
	throttle *time.Timer
}

func newAddrCache(keydir string) *addrCache {
	ac := &addrCache{
		keydir: keydir,
		byAddr: make(map[common.Address][]XPub),
	}
	ac.watcher = newWatcher(ac)
	return ac
}

func (ac *addrCache) hasAddress(addr common.Address) bool {
	ac.maybeReload()
	ac.mu.Lock()
	defer ac.mu.Unlock()
	return len(ac.byAddr[addr]) > 0
}

func (ac *addrCache) add(newKey XPub) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	i := sort.Search(len(ac.all), func(i int) bool { return ac.all[i].File >= newKey.File })
	if i < len(ac.all) && ac.all[i] == newKey {
		return
	}
	// newKey is not in the cache.
	ac.all = append(ac.all, XPub{})
	copy(ac.all[i+1:], ac.all[i:])
	ac.all[i] = newKey
	ac.byAddr[newKey.Address] = append(ac.byAddr[newKey.Address], newKey)
}

func (ac *addrCache) keys() []XPub {
	ac.maybeReload()
	ac.mu.Lock()
	defer ac.mu.Unlock()
	cpy := make([]XPub, len(ac.all))
	copy(cpy, ac.all)
	return cpy
}

func (ac *addrCache) maybeReload() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if ac.watcher.running {
		return // A watcher is running and will keep the cache up-to-date.
	}

	if ac.throttle == nil {
		ac.throttle = time.NewTimer(0)
	} else {
		select {
		case <-ac.throttle.C:
		default:
			return // The cache was reloaded recently.
		}
	}
	ac.watcher.start()
	ac.reload()
	ac.throttle.Reset(minReloadInterval)
}

// find returns the cached keys for address if there is a unique match.
// The exact matching rules are explained by the documentation of Account.
// Callers must hold ac.mu.
func (ac *addrCache) find(xpub XPub) (XPub, error) {
	// Limit search to address candidates if possible.
	matches := ac.all
	if (xpub.Address != common.Address{}) {
		matches = ac.byAddr[xpub.Address]
	}
	if xpub.File != "" {
		// If only the basename is specified, complete the path.
		if !strings.ContainsRune(xpub.File, filepath.Separator) {
			xpub.File = filepath.Join(ac.keydir, xpub.File)
		}
		for i := range matches {
			if matches[i].File == xpub.File {
				return matches[i], nil
			}
		}
		if (xpub.Address == common.Address{}) {
			return XPub{}, ErrNoKey
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return XPub{}, ErrNoKey
	default:
		err := &AmbiguousAddrError{Addr: xpub.Address, Matches: make([]XPub, len(matches))}
		copy(err.Matches, matches)
		return XPub{}, err
	}
}

// reload caches addresses of existing key.
// Callers must hold ac.mu.
func (ac *addrCache) reload() {
	keys, err := ac.scan()
	if err != nil {
		//log.Printf("can't load keys: %v", err.Error())
		fmt.Printf("can't load keys: %v\n", err.Error())

	}
	ac.all = keys
	sort.Sort(ac.all)
	for k := range ac.byAddr {
		delete(ac.byAddr, k)
	}
	for _, k := range keys {
		ac.byAddr[k.Address] = append(ac.byAddr[k.Address], k)
	}
	//log.Printf("reloaded keys, cache has %d keys", len(ac.all))
	fmt.Printf("reloaded keys, cache has %d keys\n", len(ac.all))
}

func (ac *addrCache) scan() ([]XPub, error) {
	files, err := ioutil.ReadDir(ac.keydir)
	if err != nil {
		return nil, err
	}
	var (
		buf     = new(bufio.Reader)
		keys    []XPub
		keyJSON struct {
			Address common.Address `json:"address"`
			Alias   string         `json:"alias"`
		}
	)
	for _, fi := range files {
		path := filepath.Join(ac.keydir, fi.Name())
		if skipKeyFile(fi) {
			//log.Printf("ignoring file %v", path)
			//fmt.Printf("ignoring file %v", path)
			continue
		}
		fd, err := os.Open(path)
		if err != nil {
			//log.Printf(err)
			fmt.Printf("err")
			continue
		}
		buf.Reset(fd)
		// Parse the address.
		keyJSON.Address = common.Address{}
		err = json.NewDecoder(buf).Decode(&keyJSON)
		switch {
		case err != nil:
			//log.Printf("can't decode key %s: %v", path, err)
			fmt.Printf("can't decode key %s: %v", path, err)
		case (keyJSON.Address == common.Address{}):
			fmt.Printf("can't decode key %s: missing or zero address", path)
		default:
			keys = append(keys, XPub{Address: keyJSON.Address, Alias: keyJSON.Alias, File: path})
		}
		fd.Close()
	}
	return keys, err
}

func (ac *addrCache) delete(removed XPub) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.all = removeKey(ac.all, removed)
	if ba := removeKey(ac.byAddr[removed.Address], removed); len(ba) == 0 {
		delete(ac.byAddr, removed.Address)
	} else {
		ac.byAddr[removed.Address] = ba
	}
}

func removeKey(slice []XPub, elem XPub) []XPub {
	for i := range slice {
		if slice[i] == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func skipKeyFile(fi os.FileInfo) bool {
	// Skip editor backups and UNIX-style hidden files.
	if strings.HasSuffix(fi.Name(), "~") || strings.HasPrefix(fi.Name(), ".") {
		return true
	}
	// Skip misc special files, directories (yes, symlinks too).
	if fi.IsDir() || fi.Mode()&os.ModeType != 0 {
		return true
	}
	return false
}

func (ac *addrCache) close() {
	ac.mu.Lock()
	ac.watcher.close()
	if ac.throttle != nil {
		ac.throttle.Stop()
	}
	ac.mu.Unlock()
}
