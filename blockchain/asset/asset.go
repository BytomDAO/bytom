package asset

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/golang/groupcache/lru"
	"github.com/golang/groupcache/singleflight"
	dbm "github.com/tendermint/tmlibs/db"
	"golang.org/x/crypto/sha3"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/encoding/json"
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
func Key(id *bc.AssetID) []byte {
	name := id.String()
	return []byte(assetPrefix + name)
}

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateAlias = errors.New("duplicate asset alias")
	ErrDuplicateAsset = errors.New("duplicate asset id")
	ErrSerializing    = errors.New("serializing asset definition")
	ErrMarshalAsset   = errors.New("failed marshal asset")
	ErrFindAsset      = errors.New("fail to find asset")
	ErrInternalAsset  = errors.New("btm has been defined as the internal asset")
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
	*signers.Signer
	AssetID           bc.AssetID             `json:"id"`
	Alias             *string                `json:"alias"`
	VMVersion         uint64                 `json:"vm_version"`
	IssuanceProgram   chainjson.HexBytes     `json:"issue_program"`
	InitialBlockHash  bc.Hash                `json:"init_blockhash"`
	Tags              map[string]interface{} `json:"tags"`
	RawDefinitionByte []byte                 `json:"raw_definition_byte"`
	DefinitionMap     map[string]interface{} `json:"definition"`
}

//RawDefinition return asset in the raw format
func (asset *Asset) RawDefinition() []byte {
	return asset.RawDefinitionByte
}

// Define defines a new Asset.
func (reg *Registry) Define(xpubs []chainkd.XPub, quorum int, definition map[string]interface{}, alias string, tags map[string]interface{}) (*Asset, error) {
	if alias == "btm" {
		return nil, ErrInternalAsset
	}

	if existed := reg.db.Get(aliasKey(alias)); existed != nil {
		return nil, ErrDuplicateAlias
	}

	_, assetSigner, err := signers.Create("asset", xpubs, quorum)
	if err != nil {
		return nil, err
	}

	rawDefinition, err := serializeAssetDef(definition)
	if err != nil {
		return nil, ErrSerializing
	}

	path := signers.Path(assetSigner, signers.AssetKeySpace)
	derivedXPubs := chainkd.DeriveXPubs(assetSigner.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	issuanceProgram, vmver, err := multisigIssuanceProgram(derivedPKs, assetSigner.Quorum)
	if err != nil {
		return nil, err
	}

	defHash := bc.NewHash(sha3.Sum256(rawDefinition))
	asset := &Asset{
		DefinitionMap:     definition,
		RawDefinitionByte: rawDefinition,
		VMVersion:         vmver,
		IssuanceProgram:   issuanceProgram,
		InitialBlockHash:  reg.initialBlockHash,
		AssetID:           bc.ComputeAssetID(issuanceProgram, &reg.initialBlockHash, vmver, &defHash),
		Signer:            assetSigner,
		Tags:              tags,
	}

	if existAsset := reg.db.Get(Key(&asset.AssetID)); existAsset != nil {
		return nil, ErrDuplicateAsset
	}

	if alias != "" {
		asset.Alias = &alias
	}

	ass, err := json.Marshal(asset)
	if err != nil {
		return nil, ErrMarshalAsset
	}

	storeBatch := reg.db.NewBatch()
	storeBatch.Set(aliasKey(alias), []byte(asset.AssetID.String()))
	storeBatch.Set(Key(&asset.AssetID), ass)
	storeBatch.Write()

	return asset, nil
}

// UpdateTags modifies the tags of the specified asset. The asset may be
// identified either by id or alias, but not both.
func (reg *Registry) UpdateTags(ctx context.Context, assetInfo string, tags map[string]interface{}) (err error) {
	asset := &Asset{}
	if asset, err = reg.FindByAlias(ctx, assetInfo); err != nil {
		assetID := &bc.AssetID{}
		if err := assetID.UnmarshalText([]byte(assetInfo)); err != nil {
			return err
		}
		if asset, err = reg.findByID(ctx, assetID); err != nil {
			return err
		}
	}

	asset.Tags = tags
	rawAsset, err := json.Marshal(asset)
	if err != nil {
		return ErrMarshalAsset
	}

	reg.db.Set(Key(&asset.AssetID), rawAsset)
	reg.cacheMu.Lock()
	reg.cache.Add(asset.AssetID, asset)
	reg.cacheMu.Unlock()
	return nil
}

// findByID retrieves an Asset record along with its signer, given an assetID.
func (reg *Registry) findByID(ctx context.Context, id *bc.AssetID) (*Asset, error) {
	reg.cacheMu.Lock()
	cached, ok := reg.cache.Get(id.String())
	reg.cacheMu.Unlock()
	if ok {
		return cached.(*Asset), nil
	}

	bytes := reg.db.Get(Key(id))
	if bytes == nil {
		return nil, ErrFindAsset
	}

	asset := &Asset{}
	if err := json.Unmarshal(bytes, asset); err != nil {
		return nil, err
	}

	reg.cacheMu.Lock()
	reg.cache.Add(id.String(), asset)
	reg.cacheMu.Unlock()
	return asset, nil
}

// FindByAlias retrieves an Asset record along with its signer,
// given an asset alias.
func (reg *Registry) FindByAlias(ctx context.Context, alias string) (*Asset, error) {
	reg.cacheMu.Lock()
	cachedID, ok := reg.aliasCache.Get(alias)
	reg.cacheMu.Unlock()
	if ok {
		return reg.findByID(ctx, cachedID.(*bc.AssetID))
	}

	rawID := reg.db.Get(aliasKey(alias))
	if rawID == nil {
		return nil, errors.Wrapf(ErrFindAsset, "no such asset, alias: %s", alias)
	}

	assetID := &bc.AssetID{}
	if err := assetID.UnmarshalText(rawID); err != nil {
		return nil, err
	}

	reg.cacheMu.Lock()
	reg.aliasCache.Add(alias, assetID)
	reg.cacheMu.Unlock()
	return reg.findByID(ctx, assetID)
}

func (reg *Registry) GetAliasByID(id string) string {
	if id == consensus.BTMAssetID.String() {
		return "btm"
	}
	assetID := &bc.AssetID{}
	if err := assetID.UnmarshalText([]byte(id)); err != nil {
		return ""
	}
	asset, err := reg.findByID(nil, assetID)
	if err != nil {
		return ""
	}
	return *asset.Alias
}

// ListAssets returns the accounts in the db
func (reg *Registry) ListAssets(id string) ([]*Asset, error) {
	assets := []*Asset{}
	assetIter := reg.db.IteratorPrefix([]byte(assetPrefix + id))
	defer assetIter.Release()

	for assetIter.Next() {
		asset := &Asset{}
		if err := json.Unmarshal(assetIter.Value(), asset); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
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
