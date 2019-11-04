package pseudohsm

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/cespare/cp"
	"github.com/davecgh/go-spew/spew"
)

var (
	cachetestDir, _ = filepath.Abs(filepath.Join("testdata", "keystore"))
	cachetestKeys   = []XPub{
		{
			Alias: "langyu",
			File:  filepath.Join(cachetestDir, "UTC--2017-09-13T07-11-07.863320100Z--bm1pktmny6q69dlqulja2p2ja28k2vd6wvqpk5r76a"),
		},
		{
			Alias: "aaatest",
			File:  filepath.Join(cachetestDir, "aaa"),
		},
		{
			Alias: "zzztest",
			File:  filepath.Join(cachetestDir, "zzz"),
		},
	}
)

func TestWatchNewFile(t *testing.T) {
	t.Parallel()

	dir, kc := tmpManager(t)
	// defer os.RemoveAll(dir)

	// Ensure the watcher is started before adding any files.
	kc.keys()
	time.Sleep(200 * time.Millisecond)
	// Move in the files.
	wantKeystores := make([]XPub, len(cachetestKeys))
	for i := range cachetestKeys {
		a := cachetestKeys[i]
		a.File = filepath.Join(dir, filepath.Base(a.File))
		wantKeystores[i] = a
		if err := cp.CopyFile(a.File, cachetestKeys[i].File); err != nil {
			t.Fatal(err)
		}
	}

	// kc should see the keys.
	var list []XPub
	for d := 200 * time.Millisecond; d < 5*time.Second; d *= 2 {
		list = kc.keys()
		if reflect.DeepEqual(list, wantKeystores) {
			return
		}
		time.Sleep(d)
	}
	t.Errorf("got %s, want %s", spew.Sdump(list), spew.Sdump(wantKeystores))
}

func TestWatchNoDir(t *testing.T) {
	t.Parallel()

	// Create am but not the directory that it watches.
	rand.Seed(time.Now().UnixNano())
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("bytom-keystore-watch-test-%d-%d", os.Getpid(), rand.Int()))
	kc := newKeyCache(dir)
	list := kc.keys()
	if len(list) > 0 {
		t.Error("initial account list not empty:", list)
	}
	time.Sleep(100 * time.Millisecond)

	// Create the directory and copy a key file into it.
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "aaa")
	if err := cp.CopyFile(file, cachetestKeys[0].File); err != nil {
		t.Fatal(err)
	}

	// am should see the account.
	wantKeys := []XPub{cachetestKeys[0]}
	wantKeys[0].File = file
	for d := 200 * time.Millisecond; d < 8*time.Second; d *= 2 {
		list = kc.keys()
		if reflect.DeepEqual(list, wantKeys) {
			return
		}
		time.Sleep(d)
	}
	t.Errorf("\ngot  %v\nwant %v", list, wantKeys)
}

func TestCacheInitialReload(t *testing.T) {
	cache := newKeyCache(cachetestDir)
	keys := cache.keys()
	if !reflect.DeepEqual(keys, cachetestKeys) {
		t.Fatalf("got initial accounts: %swant %s", spew.Sdump(keys), spew.Sdump(cachetestKeys))
	}
}

func TestCacheAddDeleteOrder(t *testing.T) {
	cache := newKeyCache("testdata/no-such-dir")
	cache.watcher.running = true // prevent unexpected reloads
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	keys := []XPub{
		{
			Alias: "bm1pvheagygs9d72stp79u9vduhmdyjpnvud0y89y7",
			File:  "-309830980",
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1pyk3qny8gzem6p4fx8t5d344tnldguv8lvx2aww",
			File:  "ggg",
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1p6s0ckxrudy7hqht4n5fhcs4gp69krv3c84jn9x",
			File:  "zzzzzz-the-very-last-one.keyXXX",
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1p7xkfhsw50y44t63mk0dfxxkvuyg6t3s0r6xs54",
			File:  "SOMETHING.key",
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1peu9ql7x8c7aeca60j40sg5w4kylpf7l3jmau0g",
			File:  "UTC--2016-03-22T12-57-55.920751759Z--bm1peu9ql7x8c7aeca60j40sg5w4kylpf7l3jmau0g",
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1p0s68e4ggp0vy5ue2lztsxvl2smpnqp9al8jyvh",
			File:  "aaa",
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1pjq8ttfl7ppqtcc5qqff0s45p7ew9l9pjmlu5xw",
			File:  "zzz",
			XPub:  tmpPubkeys(t, r),
		},
	}
	for _, a := range keys {
		cache.add(a)
	}
	// Add some of them twice to check that they don't get reinserted.
	cache.add(keys[0])
	cache.add(keys[2])

	// Check that the account list is sorted by filename.
	wantKeys := make([]XPub, len(keys))
	copy(wantKeys, keys)
	sort.Sort(keysByFile(wantKeys))
	list := cache.keys()

	if !reflect.DeepEqual(list, wantKeys) {
		t.Fatalf("got keys: %s\nwant %s", spew.Sdump(keys), spew.Sdump(wantKeys))
	}

	for _, a := range keys {
		if !cache.hasKey(a.XPub) {
			t.Errorf("expected hashKey(%x) to return true", a.XPub)
		}
	}
	// Delete a few keys from the cache.
	for i := 0; i < len(keys); i += 2 {
		cache.delete(wantKeys[i])
	}
	cache.delete(XPub{Alias: "bm1pug2xpcvpzepdf0paulnndhpxtpjvre8ypd0jtj", File: "something", XPub: tmpPubkeys(t, r)})

	// Check content again after deletion.
	wantKeysAfterDelete := []XPub{
		wantKeys[1],
		wantKeys[3],
		wantKeys[5],
	}
	list = cache.keys()
	if !reflect.DeepEqual(list, wantKeysAfterDelete) {
		t.Fatalf("got keys after delete: %s\nwant %s", spew.Sdump(list), spew.Sdump(wantKeysAfterDelete))
	}
	for _, a := range wantKeysAfterDelete {
		if !cache.hasKey(a.XPub) {
			t.Errorf("expected hasKey(%x) to return true", a.XPub)
		}
	}
	if cache.hasKey(wantKeys[0].XPub) {
		t.Errorf("expected hasKey(%x) to return false", wantKeys[0].XPub)
	}
}

func TestCacheFind(t *testing.T) {
	dir := filepath.Join("testdata", "dir")
	cache := newKeyCache(dir)
	cache.watcher.running = true // prevent unexpected reloads
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	dup := tmpPubkeys(t, r)
	keys := []XPub{
		{
			Alias: "bm1pmv9kg68j3edvqrv62lxllev4ugjv0zf6g5pwf6",
			File:  filepath.Join(dir, "a.key"),
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1ptspg4x6kjjp642gdpzan0ynq9zr7z4m34nqpet",
			File:  filepath.Join(dir, "b.key"),
			XPub:  tmpPubkeys(t, r),
		},
		{
			Alias: "bm1pmlpy0946zsvdg29v80gw0mkq2n0ghkg0fpmhav",
			File:  filepath.Join(dir, "c.key"),
			XPub:  dup,
		},
		{
			Alias: "bm1pmlpy0946zsvdg29v80gw0mkq2n0ghkg0fpmhav",
			File:  filepath.Join(dir, "c2.key"),
			XPub:  dup,
		},
	}
	for _, a := range keys {
		cache.add(a)
	}

	nomatchKey := XPub{
		Alias: "bm1pu2vmgps4d9e3mrsuzp58w777apky4rjgn5rn9e",
		File:  filepath.Join(dir, "something"),
		XPub:  tmpPubkeys(t, r),
	}
	tests := []struct {
		Query      XPub
		WantResult XPub
		WantError  error
	}{
		// by xpub
		{Query: XPub{XPub: keys[0].XPub}, WantResult: keys[0]},
		// by file
		{Query: XPub{File: keys[0].File}, WantResult: keys[0]},
		// by basename
		{Query: XPub{File: filepath.Base(keys[0].File)}, WantResult: keys[0]},
		// by file and xpub
		{Query: keys[0], WantResult: keys[0]},
		// ambiguous xpub, tie resolved by file
		{Query: keys[2], WantResult: keys[2]},
		// ambiguous xpub error
		{
			Query: XPub{XPub: keys[2].XPub},
			WantError: &AmbiguousKeyError{
				Pubkey:  hex.EncodeToString(keys[2].XPub[:]),
				Matches: []XPub{keys[2], keys[3]},
			},
		},
		// no match error
		{Query: nomatchKey, WantError: ErrLoadKey},
		{Query: XPub{File: nomatchKey.File}, WantError: ErrLoadKey},
		{Query: XPub{File: filepath.Base(nomatchKey.File)}, WantError: ErrLoadKey},
		{Query: XPub{XPub: nomatchKey.XPub}, WantError: ErrLoadKey},
	}
	for i, test := range tests {
		a, err := cache.find(test.Query)
		if !reflect.DeepEqual(err, test.WantError) {
			t.Errorf("test %d: error mismatch for query %v\ngot %q\nwant %q", i, test.Query, err, test.WantError)
			continue
		}
		if a != test.WantResult {
			t.Errorf("test %d: result mismatch for query %v\ngot %v\nwant %v", i, test.Query, a, test.WantResult)
			continue
		}
	}
}

func tmpManager(t *testing.T) (string, *keyCache) {
	d, err := ioutil.TempDir("", "bytom-keystore-test")
	if err != nil {
		t.Fatal(err)
	}
	return d, newKeyCache(d)
}

func tmpPubkeys(t *testing.T, r *rand.Rand) chainkd.XPub {

	var xpub chainkd.XPub
	pick := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(pick)
	result := []byte{}

	for i := 0; i < 64; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	copy(xpub[:], result[:])
	return xpub
}
