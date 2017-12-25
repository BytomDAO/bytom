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

const (
	maxAssetCache = 1000
	assetPrefix   = "ASS:"
	aliasPrefix   = "ALS:"
)

func aliasKey(name string) []byte {
	return []byte(aliasPrefix + name)
}

//Key asset store prefix
func Key(name string) []byte {
	return []byte(assetPrefix + name)
}

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateAlias = errors.New("duplicate asset alias")
	ErrBadIdentifier  = errors.New("either ID or alias must be specified, and not both")
)

//NewRegistry create new registry
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

//Asset describe asset on bytom chain
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
}

//RawDefinition return asset in the raw format
func (asset *Asset) RawDefinition() []byte {
	return asset.RawDefinitionByte
}

// SetDefinition sets the serialized asset definition.
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
	if existed := reg.db.Get(aliasKey(alias)); existed != nil {
		return nil, fmt.Errorf("%s is a duplicated account alias", alias)
	}

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

	ass, err := json.MarshalIndent(asset, "", " ")
	if err != nil {
		return nil, errors.Wrap(err, "failed marshal asset")
	}

	storeBatch := reg.db.NewBatch()
	if len(ass) > 0 {
		storeBatch.Set(aliasKey(alias), []byte(asset.AssetID.String()))
		storeBatch.Set(Key(asset.AssetID.String()), ass)
		storeBatch.Write()
	}

	return asset, nil
}

// UpdateTags modifies the tags of the specified asset. The asset may be
// identified either by id or alias, but not both.
func (reg *Registry) UpdateTags(ctx context.Context, assetInfo string, tags map[string]interface{}) error {
	var asset Asset

	assetID := assetInfo
	if s, err := reg.FindByAlias(nil, assetInfo); err == nil {
		assetID = s.AssetID.String()
	}

	rawAsset := reg.db.Get(Key(assetID))

	if rawAsset == nil {
		return errors.New("fail to find asset")
	}
	if err := json.Unmarshal(rawAsset, &asset); err != nil {
		return err
	}

	for k, v := range tags {
		switch v {
		case "":
			delete(asset.Tags, k)
		default:
			if asset.Tags == nil {
				asset.Tags = make(map[string]interface{})
			}
			asset.Tags[k] = v
		}
	}

	rawAsset, err := json.MarshalIndent(asset, "", " ")
	if err != nil {
		return errors.New("fail to marshal asset")
	}

	reg.db.Set(Key(assetID), rawAsset)
	return nil
}

// findByID retrieves an Asset record along with its signer, given an assetID.
func (reg *Registry) findByID(ctx context.Context, id string) (*Asset, error) {
	reg.cacheMu.Lock()
	cached, ok := reg.cache.Get(id)
	reg.cacheMu.Unlock()
	if ok {
		return cached.(*Asset), nil
	}

	bytes := reg.db.Get(Key(id))
	if bytes == nil {
		return nil, errors.New("no exit this asset")
	}
	var asset Asset

	if err := json.Unmarshal(bytes, &asset); err != nil {
		return nil, fmt.Errorf("err:%s,asset signer id:%s", err, id)
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
		return reg.findByID(ctx, cachedID.(string))
	}

	rawID := reg.db.Get(aliasKey(alias))
	if rawID == nil {
		return nil, fmt.Errorf("No such asset alias: %s", alias)
	}

	rawAsset := reg.db.Get(Key(string(rawID)))
	if rawAsset == nil {
		return nil, fmt.Errorf("No such asset ID: %x", rawID)
	}
	var asset Asset

	if err := json.Unmarshal(rawAsset, &asset); err != nil {
		return nil, fmt.Errorf("err:%s,asset signer id:%x", err, rawID)
	}

	reg.cacheMu.Lock()
	reg.aliasCache.Add(alias, asset.AssetID.String())
	reg.cache.Add(asset.AssetID.String(), &asset)
	reg.cacheMu.Unlock()
	return &asset, nil
}

// ListAssets returns the accounts in the db
func (reg *Registry) ListAssets() ([]Asset, error) {
	asset := Asset{}
	assets := make([]Asset, 0)

	assetIter := reg.db.IteratorPrefix([]byte(assetPrefix))
	defer assetIter.Release()

	for assetIter.Next() {
		if err := json.Unmarshal(assetIter.Value(), &asset); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}

	if len(assets) == 0 {
		return nil, errors.New("No local assets")
	}
	return assets, nil
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
