package account

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/crypto/sha3pool"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"

	chainjson "github.com/bytom/encoding/json"
)

const (
	// PinName is used to identify the pin associated with
	// the account indexer block processor.
	PinName = "account"
	// ExpirePinName is used to identify the pin associated
	// with the account control program expiration processor.
	ExpirePinName = "expire-control-programs"
	// DeleteSpentsPinName is used to identify the pin associated
	// with the processor that deletes spent account UTXOs.
	DeleteSpentsPinName = "delete-account-spents"
)

type AccountUTXOs struct {
	OutputID  []byte
	AssetID   []byte
	Amount    int64
	AccountID string
	CpIndex   int64
	Program   []byte
	Confirmed int64
	SourceID  []byte
	SourcePos int64
	RefData   []byte
	Change    bool
}

var emptyJSONObject = json.RawMessage(`{}`)

// A Saver is responsible for saving an annotated account object.
// for indexing and retrieval.
// If the Core is configured not to provide search services,
// SaveAnnotatedAccount can be a no-op.
type Saver interface {
	SaveAnnotatedAccount(context.Context, *query.AnnotatedAccount) error
}

func Annotated(a *Account) (*query.AnnotatedAccount, error) {
	aa := &query.AnnotatedAccount{
		ID:     a.ID,
		Alias:  a.Alias,
		Quorum: a.Quorum,
		Tags:   &emptyJSONObject,
	}

	tags, err := json.Marshal(a.Tags)
	if err != nil {
		return nil, err
	}
	if len(tags) > 0 {
		rawTags := json.RawMessage(tags)
		aa.Tags = &rawTags
	}

	path := signers.Path(a.Signer, signers.AccountKeySpace)
	var jsonPath []chainjson.HexBytes
	for _, p := range path {
		jsonPath = append(jsonPath, p)
	}
	for _, xpub := range a.XPubs {
		aa.Keys = append(aa.Keys, &query.AccountKey{
			RootXPub:              xpub,
			AccountXPub:           xpub.Derive(path),
			AccountDerivationPath: jsonPath,
		})
	}
	return aa, nil
}

func (m *Manager) indexAnnotatedAccount(ctx context.Context, a *Account) error {
	if m.indexer == nil {
		return nil
	}
	aa, err := Annotated(a)
	if err != nil {
		return err
	}
	return m.indexer.SaveAnnotatedAccount(ctx, aa)
}

type rawOutput struct {
	OutputID bc.Hash
	bc.AssetAmount
	ControlProgram []byte
	txHash         bc.Hash
	outputIndex    uint32
	sourceID       bc.Hash
	sourcePos      uint64
	refData        bc.Hash
}

type accountOutput struct {
	rawOutput
	AccountID string
	keyIndex  uint64
	change    bool
}

func (m *Manager) ProcessBlocks(ctx context.Context) {
	if m.pinStore == nil {
		return
	}

	go m.pinStore.ProcessBlocks(ctx, m.chain, DeleteSpentsPinName, func(ctx context.Context, b *legacy.Block) error {
		<-m.pinStore.PinWaiter(PinName, b.Height)
		return m.deleteSpentOutputs(ctx, b)
	})
	m.pinStore.ProcessBlocks(ctx, m.chain, PinName, m.indexAccountUTXOs)

}

func (m *Manager) deleteSpentOutputs(ctx context.Context, b *legacy.Block) error {
	// Delete consumed account UTXOs.
	delOutputIDs := prevoutDBKeys(b.Transactions...)
	for _, delOutputID := range delOutputIDs {
		m.pinStore.DB.Delete(json.RawMessage("acu" + string(delOutputID.Bytes())))
	}

	return errors.Wrap(nil, "deleting spent account utxos")
}

func (m *Manager) indexAccountUTXOs(ctx context.Context, b *legacy.Block) error {
	// Upsert any UTXOs belonging to accounts managed by this Core.
	outs := make([]*rawOutput, 0, len(b.Transactions))
	blockPositions := make(map[bc.Hash]uint32, len(b.Transactions))
	for i, tx := range b.Transactions {
		blockPositions[tx.ID] = uint32(i)
		for j, out := range tx.Outputs {
			resOutID := tx.ResultIds[j]
			resOut, ok := tx.Entries[*resOutID].(*bc.Output)
			if !ok {
				continue
			}
			out := &rawOutput{
				OutputID:       *tx.OutputID(j),
				AssetAmount:    out.AssetAmount,
				ControlProgram: out.ControlProgram,
				txHash:         tx.ID,
				outputIndex:    uint32(j),
				sourceID:       *resOut.Source.Ref,
				sourcePos:      resOut.Source.Position,
				refData:        *resOut.Data,
			}
			outs = append(outs, out)
		}
	}
	accOuts := m.loadAccountInfo(ctx, outs)

	err := m.upsertConfirmedAccountOutputs(ctx, accOuts, blockPositions, b)
	return errors.Wrap(err, "upserting confirmed account utxos")
}

func prevoutDBKeys(txs ...*legacy.Tx) (outputIDs []bc.Hash) {
	for _, tx := range txs {
		for _, inpID := range tx.Tx.InputIDs {
			if sp, err := tx.Spend(inpID); err == nil {
				outputIDs = append(outputIDs, *sp.SpentOutputId)
			}
		}
	}
	return
}

// loadAccountInfo turns a set of output IDs into a set of
// outputs by adding account annotations.  Outputs that can't be
// annotated are excluded from the result.
func (m *Manager) loadAccountInfo(ctx context.Context, outs []*rawOutput) []*accountOutput {
	outsByScript := make(map[string][]*rawOutput, len(outs))
	for _, out := range outs {
		scriptStr := string(out.ControlProgram)
		outsByScript[scriptStr] = append(outsByScript[scriptStr], out)
	}

	result := make([]*accountOutput, 0, len(outs))
	cp := struct {
		AccountID      string
		KeyIndex       uint64
		ControlProgram []byte
		Change         bool
		ExpiresAt      time.Time
	}{}

	var b32 [32]byte
	for s := range outsByScript {
		sha3pool.Sum256(b32[:], []byte(s))
		bytes := m.db.Get(json.RawMessage("acp" + string(b32[:])))
		if bytes == nil {
			continue
		}

		err := json.Unmarshal(bytes, &cp)
		if err != nil {
			continue
		}

		//filte the accounts which exists in accountdb with wallet enabled
		isExist := m.db.Get(json.RawMessage(cp.AccountID))
		if isExist == nil {
			continue
		}

		for _, out := range outsByScript[s] {
			newOut := &accountOutput{
				rawOutput: *out,
				AccountID: cp.AccountID,
				keyIndex:  cp.KeyIndex,
				change:    cp.Change,
			}
			result = append(result, newOut)
		}
	}

	return result
}

// upsertConfirmedAccountOutputs records the account data for confirmed utxos.
// If the account utxo already exists (because it's from a local tx), the
// block confirmation data will in the row will be updated.
func (m *Manager) upsertConfirmedAccountOutputs(ctx context.Context,
	outs []*accountOutput,
	pos map[bc.Hash]uint32,
	block *legacy.Block) error {

	var au *AccountUTXOs
	for _, out := range outs {
		au = &AccountUTXOs{OutputID: out.OutputID.Bytes(),
			AssetID:   out.AssetId.Bytes(),
			Amount:    int64(out.Amount),
			AccountID: out.AccountID,
			CpIndex:   int64(out.keyIndex),
			Program:   out.ControlProgram,
			Confirmed: int64(block.Height),
			SourceID:  out.sourceID.Bytes(),
			SourcePos: int64(out.sourcePos),
			RefData:   out.refData.Bytes(),
			Change:    out.change}

		accountutxo, err := json.Marshal(au)
		if err != nil {
			return errors.Wrap(err, "failed marshal accountutxo")
		}

		if len(accountutxo) > 0 {
			m.pinStore.DB.Set(json.RawMessage("acu"+string(au.OutputID)), accountutxo)
		}

	}

	return nil
}
