package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"
	dbm "github.com/tendermint/tmlibs/db"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/vm/vmutil"
)

const maxAssetCache = 1000

var (
	ErrDuplicateAlias = errors.New("duplicate asset alias")
	ErrBadIdentifier  = errors.New("either ID or alias must be specified, and not both")
)

func NewRegistry(db dbm.DB, chain *protocol.Chain) *Registry {
	return &Registry{
		db:               db,
		chain:            chain,
		initialBlockHash: chain.InitialBlockHash,
		cache:            lru.New(maxAssetCache),
		aliasCache:       lru.New(maxAssetCache),
	}
}

// Registry tracks and stores all known assets on a blockchain.
type Registry struct {
	db               dbm.DB
	chain            *protocol.Chain
	initialBlockHash bc.Hash

	idGroup    singleflight.Group
	aliasGroup singleflight.Group

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache
}

type Asset struct {
	AssetID          bc.AssetID
	Alias            *string
	VMVersion        uint64
	IssuanceProgram  []byte
	InitialBlockHash bc.Hash
	*signers.Signer
	Tags              map[string]interface{}
	RawDefinitionByte []byte
	DefinitionMap     map[string]interface{}
	BlockHeight       uint64
}

func (asset *Asset) Definition() (map[string]interface{}, error) {
	if asset.DefinitionMap == nil && len(asset.RawDefinitionByte) > 0 {
		err := json.Unmarshal(asset.RawDefinitionByte, &asset.DefinitionMap)
		if err != nil {
			return nil, errors.Wrap(err)
		}
	}
	return asset.DefinitionMap, nil
}

func (asset *Asset) RawDefinition() []byte {
	return asset.RawDefinitionByte
}

func (asset *Asset) SetDefinition(def map[string]interface{}) error {
	rawdef, err := serializeAssetDef(def)
	if err != nil {
		return err
	}
	asset.DefinitionMap = def
	asset.RawDefinitionByte = rawdef
	return nil
}

// Define defines a new Asset.
func (reg *Registry) Define(ctx context.Context, xpubs []chainkd.XPub, quorum int, definition map[string]interface{}, alias string, tags map[string]interface{}, clientToken string) (*Asset, error) {
	assetSigner, err := signers.Create(ctx, reg.db, "asset", xpubs, quorum, clientToken)
	if err != nil {
		return nil, err
	}

	rawDefinition, err := serializeAssetDef(definition)
	if err != nil {
		return nil, errors.Wrap(err, "serializing asset definition")
	}

	path := signers.Path(assetSigner, signers.AssetKeySpace)
	derivedXPubs := chainkd.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	issuanceProgram, vmver, err := multisigIssuanceProgram(derivedPKs, assetSigner.Quorum)
	if err != nil {
		return nil, err
	}

	defhash := bc.NewHash(sha3.Sum256(rawDefinition))
	asset := &Asset{
		DefinitionMap:     definition,
		RawDefinitionByte: rawDefinition,
		VMVersion:         vmver,
		IssuanceProgram:   issuanceProgram,
		InitialBlockHash:  reg.initialBlockHash,
		AssetID:           bc.ComputeAssetID(issuanceProgram, &reg.initialBlockHash, vmver, &defhash),
		Signer:            assetSigner,
		Tags:              tags,
	}
	if alias != "" {
		asset.Alias = &alias
	}

	assetID := []byte(asset.AssetID.String())
	ass, err := json.Marshal(asset)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshal asset")
	}
	if len(ass) > 0 {
		reg.db.Set(assetID, json.RawMessage(ass))
	}

	return asset, nil
}

// UpdateTags modifies the tags of the specified asset. The asset may be
// identified either by id or alias, but not both.

func (reg *Registry) UpdateTags(ctx context.Context, id, alias *string, tags map[string]interface{}) error {
	if (id == nil) == (alias == nil) {
		return errors.Wrap(ErrBadIdentifier)
	}

	// Fetch the existing asset

	var (
		asset *Asset
		err   error
	)

	if id != nil {
		var aid bc.AssetID
		err = aid.UnmarshalText([]byte(*id))
		if err != nil {
			return errors.Wrap(err, "deserialize asset ID")
		}

		asset, err = reg.findByID(ctx, aid)
		if err != nil {
			return errors.Wrap(err, "find asset by ID")
		}
	} else {
		return nil
		asset, err = reg.FindByAlias(ctx, *alias)
		if err != nil {
			return errors.Wrap(err, "find asset by alias")
		}
	}

	// Revise tags in-memory

	asset.Tags = tags

	reg.cacheMu.Lock()
	reg.cache.Add(asset.AssetID, asset)
	reg.cacheMu.Unlock()

	return nil

}

// findByID retrieves an Asset record along with its signer, given an assetID.
func (reg *Registry) findByID(ctx context.Context, id bc.AssetID) (*Asset, error) {
	reg.cacheMu.Lock()
	cached, ok := reg.cache.Get(id)
	reg.cacheMu.Unlock()
	if ok {
		return cached.(*Asset), nil
	}

	bytes := reg.db.Get([]byte(id.String()))
	if bytes == nil {
		return nil, errors.New("no exit this asset")
	}
	var asset Asset

	if err := json.Unmarshal(bytes, &asset); err != nil {
		return nil, fmt.Errorf("err:%s,asset signer id:%s", err, id.String())
	}

	reg.cacheMu.Lock()
	reg.cache.Add(id, &asset)
	reg.cacheMu.Unlock()
	return &asset, nil
}

// FindByAlias retrieves an Asset record along with its signer,
// given an asset alias.

func (reg *Registry) FindByAlias(ctx context.Context, alias string) (*Asset, error) {
	reg.cacheMu.Lock()
	cachedID, ok := reg.aliasCache.Get(alias)
	reg.cacheMu.Unlock()
	if ok {
		return reg.findByID(ctx, cachedID.(bc.AssetID))
	}

	untypedAsset, err := reg.aliasGroup.Do(alias, func() (interface{}, error) {
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	a := untypedAsset.(*Asset)
	reg.cacheMu.Lock()
	reg.aliasCache.Add(alias, a.AssetID)
	reg.cache.Add(a.AssetID, a)
	reg.cacheMu.Unlock()
	return a, nil

}

func (reg *Registry) QueryAll(ctx context.Context) (interface{}, error) {
	ret := make([]interface{}, 0)

	assetIter := reg.db.Iterator()
	defer assetIter.Release()

	for assetIter.Next() {
		value := string(assetIter.Value())
		ret = append(ret, value)
	}

	return ret, nil
}

// serializeAssetDef produces a canonical byte representation of an asset
// definition. Currently, this is implemented using pretty-printed JSON.
// As is the standard for Go's map[string] serialization, object keys will
// appear in lexicographic order. Although this is mostly meant for machine
// consumption, the JSON is pretty-printed for easy reading.
// The empty asset def is an empty byte slice.
func serializeAssetDef(def map[string]interface{}) ([]byte, error) {
	if def == nil {
		return []byte{}, nil
	}
	return json.MarshalIndent(def, "", "  ")
}

func multisigIssuanceProgram(pubkeys []ed25519.PublicKey, nrequired int) (program []byte, vmversion uint64, err error) {
	issuanceProg, err := vmutil.P2SPMultiSigProgram(pubkeys, nrequired)
	if err != nil {
		return nil, 0, err
	}
	builder := vmutil.NewBuilder()
	builder.AddRawBytes(issuanceProg)
	prog, err := builder.Build()
	return prog, 1, err
}
