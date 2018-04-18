package asset

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"strings"
	"sync"

	"github.com/golang/groupcache/lru"
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

// DefaultNativeAsset native BTM asset
var DefaultNativeAsset *Asset

const (
	maxAssetCache = 1000
	assetPrefix   = "ASS:"
	//AliasPrefix is asset alias prefix
	AliasPrefix = "ALS:"
	//ExternalAssetPrefix is external definition assets prefix
	ExternalAssetPrefix = "EXA"
)

var assetIndexKey = []byte("ASSETINDEX")

func initNativeAsset() {
	signer := &signers.Signer{Type: "internal"}
	alias := consensus.BTMAlias

	definitionBytes, _ := serializeAssetDef(consensus.BTMDefinitionMap)
	DefaultNativeAsset = &Asset{
		Signer:            signer,
		AssetID:           *consensus.BTMAssetID,
		Alias:             &alias,
		VMVersion:         1,
		DefinitionMap:     consensus.BTMDefinitionMap,
		RawDefinitionByte: definitionBytes,
	}
}

// AliasKey store asset alias prefix
func AliasKey(name string) []byte {
	return []byte(AliasPrefix + name)
}

//Key asset store prefix
func Key(id *bc.AssetID) []byte {
	name := id.String()
	return []byte(assetPrefix + name)
}

//CalcExtAssetKey return store external assets key
func CalcExtAssetKey(id *bc.AssetID) []byte {
	name := id.String()
	return []byte(ExternalAssetPrefix + name)
}

// pre-define errors for supporting bytom errorFormatter
var (
	ErrDuplicateAlias = errors.New("duplicate asset alias")
	ErrDuplicateAsset = errors.New("duplicate asset id")
	ErrSerializing    = errors.New("serializing asset definition")
	ErrMarshalAsset   = errors.New("failed marshal asset")
	ErrFindAsset      = errors.New("fail to find asset")
	ErrInternalAsset  = errors.New("btm has been defined as the internal asset")
	ErrNullAlias      = errors.New("null asset alias")
)

//NewRegistry create new registry
func NewRegistry(db dbm.DB, chain *protocol.Chain) *Registry {
	initNativeAsset()
	return &Registry{
		db:         db,
		chain:      chain,
		cache:      lru.New(maxAssetCache),
		aliasCache: lru.New(maxAssetCache),
	}
}

// Registry tracks and stores all known assets on a blockchain.
type Registry struct {
	db    dbm.DB
	chain *protocol.Chain

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache

	assetIndexMu sync.Mutex
}

//Asset describe asset on bytom chain
type Asset struct {
	*signers.Signer
	AssetID           bc.AssetID             `json:"id"`
	Alias             *string                `json:"alias"`
	VMVersion         uint64                 `json:"vm_version"`
	IssuanceProgram   chainjson.HexBytes     `json:"issue_program"`
	RawDefinitionByte chainjson.HexBytes     `json:"raw_definition_byte"`
	DefinitionMap     map[string]interface{} `json:"definition"`
}

func (reg *Registry) getNextAssetIndex() (uint64, error) {
	reg.assetIndexMu.Lock()
	defer reg.assetIndexMu.Unlock()

	var nextIndex uint64 = 1

	if rawIndex := reg.db.Get(assetIndexKey); rawIndex != nil {
		nextIndex = binary.LittleEndian.Uint64(rawIndex) + 1
	}

	reg.saveNextAssetIndex(nextIndex)
	return nextIndex, nil
}

func (reg *Registry) saveNextAssetIndex(nextIndex uint64) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, nextIndex)
	reg.db.Set(assetIndexKey, buf)
}

// Define defines a new Asset.
func (reg *Registry) Define(xpubs []chainkd.XPub, quorum int, definition map[string]interface{}, alias string) (*Asset, error) {
	if len(xpubs) == 0 {
		return nil, errors.Wrap(signers.ErrNoXPubs)
	}

	normalizedAlias := strings.ToUpper(strings.TrimSpace(alias))
	if normalizedAlias == consensus.BTMAlias {
		return nil, ErrInternalAsset
	}

	if existed := reg.db.Get(AliasKey(normalizedAlias)); existed != nil {
		return nil, ErrDuplicateAlias
	}

	nextAssetIndex, err := reg.getNextAssetIndex()
	if err != nil {
		return nil, errors.Wrap(err, "get asset index error")
	}

	assetSigner, err := signers.Create("asset", xpubs, quorum, nextAssetIndex)
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
		AssetID:           bc.ComputeAssetID(issuanceProgram, vmver, &defHash),
		Signer:            assetSigner,
	}

	if existAsset := reg.db.Get(Key(&asset.AssetID)); existAsset != nil {
		return nil, ErrDuplicateAsset
	}

	if alias != "" {
		asset.Alias = &normalizedAlias
	}

	ass, err := json.Marshal(asset)
	if err != nil {
		return nil, ErrMarshalAsset
	}

	storeBatch := reg.db.NewBatch()
	storeBatch.Set(AliasKey(normalizedAlias), []byte(asset.AssetID.String()))
	storeBatch.Set(Key(&asset.AssetID), ass)
	storeBatch.Write()

	return asset, nil
}

// FindByID retrieves an Asset record along with its signer, given an assetID.
func (reg *Registry) FindByID(ctx context.Context, id *bc.AssetID) (*Asset, error) {
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

//GetIDByAlias return AssetID string and nil by asset alias,if err ,return "" and err
func (reg *Registry) GetIDByAlias(alias string) (string, error) {
	rawID := reg.db.Get(AliasKey(alias))
	if rawID == nil {
		return "", ErrFindAsset
	}
	return string(rawID), nil
}

// FindByAlias retrieves an Asset record along with its signer,
// given an asset alias.
func (reg *Registry) FindByAlias(ctx context.Context, alias string) (*Asset, error) {
	reg.cacheMu.Lock()
	cachedID, ok := reg.aliasCache.Get(alias)
	reg.cacheMu.Unlock()
	if ok {
		return reg.FindByID(ctx, cachedID.(*bc.AssetID))
	}

	rawID := reg.db.Get(AliasKey(alias))
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
	return reg.FindByID(ctx, assetID)
}

//GetAliasByID return asset alias string by AssetID string
func (reg *Registry) GetAliasByID(id string) string {
	//btm
	if id == consensus.BTMAssetID.String() {
		return consensus.BTMAlias
	}

	assetID := &bc.AssetID{}
	if err := assetID.UnmarshalText([]byte(id)); err != nil {
		return ""
	}

	asset, err := reg.FindByID(nil, assetID)
	if err != nil {
		return ""
	}

	return *asset.Alias
}

// ListAssets returns the accounts in the db
func (reg *Registry) ListAssets(id string) ([]*Asset, error) {
	assets := []*Asset{DefaultNativeAsset}
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
func serializeAssetDef(def map[string]interface{}) ([]byte, error) {
	if def == nil {
		def = make(map[string]interface{}, 0)
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

//UpdateAssetAlias updates asset alias
func (reg *Registry) UpdateAssetAlias(id, newAlias string) error {
	oldAlias := reg.GetAliasByID(id)
	normalizedAlias := strings.ToUpper(strings.TrimSpace(newAlias))

	if oldAlias == consensus.BTMAlias || newAlias == consensus.BTMAlias {
		return ErrInternalAsset
	}

	if oldAlias == "" || normalizedAlias == "" {
		return ErrNullAlias
	}

	if _, err := reg.GetIDByAlias(normalizedAlias); err == nil {
		return ErrDuplicateAlias
	}

	findAsset, err := reg.FindByAlias(nil, oldAlias)
	if err != nil {
		return err
	}

	storeBatch := reg.db.NewBatch()
	findAsset.Alias = &normalizedAlias
	assetID := &findAsset.AssetID
	rawAsset, err := json.Marshal(findAsset)
	if err != nil {
		return err
	}

	storeBatch.Set(Key(assetID), rawAsset)
	storeBatch.Set(AliasKey(newAlias), []byte(assetID.String()))
	storeBatch.Delete(AliasKey(oldAlias))
	storeBatch.Write()

	reg.cacheMu.Lock()
	reg.aliasCache.Add(newAlias, assetID)
	reg.aliasCache.Remove(oldAlias)
	reg.cacheMu.Unlock()

	return nil
}
