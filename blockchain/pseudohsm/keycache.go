package pseudohsm

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
)

// Minimum amount of time between cache reloads. This limit applies if the platform does
// not support change notifications. It also applies if the keystore directory does not
// exist yet, the code will attempt to create a watcher at most this often.
const minReloadInterval = 2 * time.Second

type keysByFile []XPub

func (s keysByFile) Len() int           { return len(s) }
func (s keysByFile) Less(i, j int) bool { return s[i].File < s[j].File }
func (s keysByFile) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// AmbiguousKeyError is returned when attempting to unlock
// an XPub for which more than one file exists.
type AmbiguousKeyError struct {
	Pubkey  string
	Matches []XPub
}

func (err *AmbiguousKeyError) Error() string {
	files := ""
	for i, a := range err.Matches {
		files += a.File
		if i < len(err.Matches)-1 {
			files += ", "
		}
	}
	return fmt.Sprintf("multiple keys match keys (%s)", files)
}

// keyCache is a live index of all keys in the keystore.
type keyCache struct {
	keydir   string
	watcher  *watcher
	mu       sync.Mutex
	all      keysByFile
	byPubs   map[chainkd.XPub][]XPub
	throttle *time.Timer
}

func newKeyCache(keydir string) *keyCache {
	kc := &keyCache{
		keydir: keydir,
		byPubs: make(map[chainkd.XPub][]XPub),
	}
	kc.watcher = newWatcher(kc)
	return kc
}

func (kc *keyCache) hasKey(xpub chainkd.XPub) bool {
	kc.maybeReload()
	kc.mu.Lock()
	defer kc.mu.Unlock()
	return len(kc.byPubs[xpub]) > 0
}

func (kc *keyCache) hasAlias(alias string) bool {
	xpubs := kc.keys()
	for _, xpub := range xpubs {
		if xpub.Alias == alias {
			return true
		}
	}
	return false
}

func (kc *keyCache) add(newKey XPub) {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	i := sort.Search(len(kc.all), func(i int) bool { return kc.all[i].File >= newKey.File })
	if i < len(kc.all) && kc.all[i] == newKey {
		return
	}
	// newKey is not in the cache.
	kc.all = append(kc.all, XPub{})
	copy(kc.all[i+1:], kc.all[i:])
	kc.all[i] = newKey
	kc.byPubs[newKey.XPub] = append(kc.byPubs[newKey.XPub], newKey)
}

func (kc *keyCache) keys() []XPub {
	kc.maybeReload()
	kc.mu.Lock()
	defer kc.mu.Unlock()
	cpy := make([]XPub, len(kc.all))
	copy(cpy, kc.all)
	return cpy
}

func (kc *keyCache) maybeReload() {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.watcher.running {
		return // A watcher is running and will keep the cache up-to-date.
	}

	if kc.throttle == nil {
		kc.throttle = time.NewTimer(0)
	} else {
		select {
		case <-kc.throttle.C:
		default:
			return // The cache was reloaded recently.
		}
	}
	kc.watcher.start()
	kc.reload()
	kc.throttle.Reset(minReloadInterval)
}

// find returns the cached keys for alias if there is a unique match.
// The exact matching rules are explained by the documentation of Account.
// Callers must hold ac.mu.
func (kc *keyCache) find(xpub XPub) (XPub, error) {
	// Limit search to xpub candidates if possible.
	matches := kc.all
	if (xpub.XPub != chainkd.XPub{}) {
		matches = kc.byPubs[xpub.XPub]
	}
	if xpub.File != "" {
		// If only the basename is specified, complete the path.
		if !strings.ContainsRune(xpub.File, filepath.Separator) {
			xpub.File = filepath.Join(kc.keydir, xpub.File)
		}
		for i := range matches {
			if matches[i].File == xpub.File {
				return matches[i], nil
			}
		}
		if (xpub.XPub == chainkd.XPub{}) {
			return XPub{}, ErrLoadKey
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return XPub{}, ErrLoadKey
	default:
		err := &AmbiguousKeyError{Pubkey: hex.EncodeToString(xpub.XPub[:]), Matches: make([]XPub, len(matches))}
		copy(err.Matches, matches)
		return XPub{}, err
	}
}

// reload caches addresses of existing key.
// Callers must hold ac.mu.
func (kc *keyCache) reload() {
	keys, err := kc.scan()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "load keys error": err}).Error("can't load keys")
	}
	kc.all = keys
	sort.Sort(kc.all)
	for k := range kc.byPubs {
		delete(kc.byPubs, k)
	}
	for _, k := range keys {
		kc.byPubs[k.XPub] = append(kc.byPubs[k.XPub], k)
	}
	log.WithFields(log.Fields{"module": logModule, "cache has keys:": len(kc.all)}).Debug("reloaded keys")
}

func (kc *keyCache) scan() ([]XPub, error) {
	files, err := ioutil.ReadDir(kc.keydir)
	if err != nil {
		return nil, err
	}
	var (
		buf     = new(bufio.Reader)
		keys    []XPub
		keyJSON struct {
			Alias string       `json:"alias"`
			XPub  chainkd.XPub `json:"xpub"`
		}
	)
	for _, fi := range files {
		path := filepath.Join(kc.keydir, fi.Name())
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
		keyJSON.Alias = ""
		err = json.NewDecoder(buf).Decode(&keyJSON)
		switch {
		case err != nil:
			log.WithFields(log.Fields{"module": logModule, "decode json err": err}).Errorf("can't decode key %s: %v", path, err)

		case (keyJSON.Alias == ""):
			log.WithFields(log.Fields{"module": logModule, "can't decode key, key path:": path}).Warn("missing or void alias")
		default:
			keys = append(keys, XPub{XPub: keyJSON.XPub, Alias: keyJSON.Alias, File: path})
		}
		fd.Close()
	}
	return keys, err
}

func (kc *keyCache) delete(removed XPub) {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	kc.all = removeKey(kc.all, removed)
	if ba := removeKey(kc.byPubs[removed.XPub], removed); len(ba) == 0 {
		delete(kc.byPubs, removed.XPub)
	} else {
		kc.byPubs[removed.XPub] = ba
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

func (kc *keyCache) close() {
	kc.mu.Lock()
	kc.watcher.close()
	if kc.throttle != nil {
		kc.throttle.Stop()
	}
	kc.mu.Unlock()
}
