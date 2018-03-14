package query

import (
	"encoding/json"
	"time"

	"github.com/bytom/crypto/ed25519/chainkd"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/bc"
)

//AnnotatedTx means an annotated transaction.
type AnnotatedTx struct {
	ID                     bc.Hash             `json:"id"`
	Timestamp              time.Time           `json:"timestamp"`
	BlockID                bc.Hash             `json:"block_id"`
	BlockHeight            uint64              `json:"block_height"`
	Position               uint32              `json:"position"`
	BlockTransactionsCount uint32              `json:"block_transactions_count,omitempty"`
	Inputs                 []*AnnotatedInput   `json:"inputs"`
	Outputs                []*AnnotatedOutput  `json:"outputs"`
	StatusFail             bool                `json:"status_fail"`
}

//AnnotatedInput means an annotated transaction input.
type AnnotatedInput struct {
	Type            string              `json:"type"`
	AssetID         bc.AssetID          `json:"asset_id"`
	AssetAlias      string              `json:"asset_alias,omitempty"`
	AssetDefinition *chainjson.HexBytes `json:"asset_definition"`
	Amount          uint64              `json:"amount"`
	IssuanceProgram chainjson.HexBytes  `json:"issuance_program,omitempty"`
	ControlProgram  chainjson.HexBytes  `json:"-"`
	SpentOutputID   *bc.Hash            `json:"spent_output_id,omitempty"`
	AccountID       string              `json:"account_id,omitempty"`
	AccountAlias    string              `json:"account_alias,omitempty"`
	Arbitrary       chainjson.HexBytes  `json:"arbitrary,omitempty"`
}

//AnnotatedOutput means an annotated transaction output.
type AnnotatedOutput struct {
	Type            string              `json:"type"`
	OutputID        bc.Hash             `json:"id"`
	TransactionID   *bc.Hash            `json:"transaction_id,omitempty"`
	Position        int                 `json:"position"`
	AssetID         bc.AssetID          `json:"asset_id"`
	AssetAlias      string              `json:"asset_alias,omitempty"`
	AssetDefinition *chainjson.HexBytes `json:"asset_definition"`
	Amount          uint64              `json:"amount"`
	AccountID       string              `json:"account_id,omitempty"`
	AccountAlias    string              `json:"account_alias,omitempty"`
	ControlProgram  chainjson.HexBytes  `json:"control_program"`
}

//AnnotatedAccount means an annotated account.
type AnnotatedAccount struct {
	ID     string           `json:"id"`
	Alias  string           `json:"alias,omitempty"`
	Keys   []*AccountKey    `json:"keys"`
	Quorum int              `json:"quorum"`
	Tags   *json.RawMessage `json:"tags"`
}

//AccountKey means an account key.
type AccountKey struct {
	RootXPub              chainkd.XPub         `json:"root_xpub"`
	AccountXPub           chainkd.XPub         `json:"account_xpub"`
	AccountDerivationPath []chainjson.HexBytes `json:"account_derivation_path"`
}

//AnnotatedAsset means an annotated asset.
type AnnotatedAsset struct {
	ID              bc.AssetID         `json:"id"`
	Alias           string             `json:"alias,omitempty"`
	IssuanceProgram chainjson.HexBytes `json:"issuance_program"`
	Keys            []*AssetKey        `json:"keys"`
	Quorum          int                `json:"quorum"`
	Definition      *json.RawMessage   `json:"definition"`
	Tags            *json.RawMessage   `json:"tags"`
}

//AssetKey means an asset key.
type AssetKey struct {
	RootXPub            chainkd.XPub         `json:"root_xpub"`
	AssetPubkey         chainjson.HexBytes   `json:"asset_pubkey"`
	AssetDerivationPath []chainjson.HexBytes `json:"asset_derivation_path"`
}
