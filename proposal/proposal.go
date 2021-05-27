package proposal

import (
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/blockchain/txbuilder"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
	"github.com/bytom/bytom/protocol/validation"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

const (
	logModule     = "proposal"
	batchApplyNum = 16
	softMaxTxNum  = 128

	timeoutOk = iota + 1
	timeoutWarn
	timeoutCritical
)

// NewBlockTemplate returns a new block template that is ready to be solved
func NewBlockTemplate(chain *protocol.Chain, validator *state.Validator, accountManager *account.Manager, timestamp uint64, warnDuration, criticalDuration time.Duration) (*types.Block, error) {
	builder := newBlockBuilder(chain, validator, accountManager, timestamp, warnDuration, criticalDuration)
	return builder.build()
}

type blockBuilder struct {
	chain          *protocol.Chain
	validator      *state.Validator
	accountManager *account.Manager

	block    *types.Block
	utxoView *state.UtxoViewpoint

	warnTimeoutCh     <-chan time.Time
	criticalTimeoutCh <-chan time.Time
	timeoutStatus     uint8
	gasLeft           int64
}

func newBlockBuilder(chain *protocol.Chain, validator *state.Validator, accountManager *account.Manager, timestamp uint64, warnDuration, criticalDuration time.Duration) *blockBuilder {
	preBlockHeader := chain.BestBlockHeader()
	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            preBlockHeader.Height + 1,
			PreviousBlockHash: preBlockHeader.Hash(),
			Timestamp:         timestamp,
			BlockCommitment:   types.BlockCommitment{},
		},
	}

	builder := &blockBuilder{
		chain:             chain,
		validator:         validator,
		accountManager:    accountManager,
		block:             block,
		utxoView:          state.NewUtxoViewpoint(),
		warnTimeoutCh:     time.After(warnDuration),
		criticalTimeoutCh: time.After(criticalDuration),
		gasLeft:           int64(consensus.MaxBlockGas),
		timeoutStatus:     timeoutOk,
	}
	return builder
}

func (b *blockBuilder) build() (*types.Block, error) {
	b.block.Transactions = []*types.Tx{nil}
	feeAmount, err := b.applyTransactionFromPool()
	if err != nil {
		return nil, err
	}

	if err := b.applyCoinbaseTransaction(feeAmount); err != nil {
		return nil, err
	}

	if err := b.calculateBlockCommitment(); err != nil {
		return nil, err
	}

	blockHeader := &b.block.BlockHeader
	b.chain.SignBlockHeader(blockHeader)
	return b.block, nil
}

func (b *blockBuilder) applyCoinbaseTransaction(feeAmount uint64) error {
	coinbaseTx, err := b.createCoinbaseTx(feeAmount)
	if err != nil {
		return errors.Wrap(err, "fail on create coinbase tx")
	}

	gasState, err := validation.ValidateTx(coinbaseTx.Tx, &bc.Block{BlockHeader: &bc.BlockHeader{Height: b.block.Height}, Transactions: []*bc.Tx{coinbaseTx.Tx}}, b.chain.ProgramConverter)
	if err != nil {
		return err
	}

	b.block.Transactions[0] = coinbaseTx
	b.gasLeft -= gasState.GasUsed
	return nil
}

func (b *blockBuilder) applyTransactionFromPool() (uint64, error) {
	txDescList := b.chain.GetTxPool().GetTransactions()
	sort.Sort(byTime(txDescList))
	return b.applyTransactions(txDescList, timeoutWarn)
}

func (b *blockBuilder) calculateBlockCommitment() (err error) {
	var txEntries []*bc.Tx
	for _, tx := range b.block.Transactions {
		txEntries = append(txEntries, tx.Tx)
	}

	b.block.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	if err != nil {
		return err
	}

	return nil
}

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
func (b *blockBuilder) createCoinbaseTx(feeAmount uint64) (tx *types.Tx, err error) {
	arbitrary := append([]byte{0x00}, []byte(strconv.FormatUint(b.block.Height, 10))...)
	var script []byte
	if b.accountManager == nil {
		script, err = vmutil.DefaultCoinbaseProgram()
	} else {
		script, err = b.accountManager.GetCoinbaseControlProgram()
		arbitrary = append(arbitrary, b.accountManager.GetCoinbaseArbitrary()...)
	}
	if err != nil {
		return nil, err
	}

	if len(arbitrary) > consensus.CoinbaseArbitrarySizeLimit {
		return nil, validation.ErrCoinbaseArbitraryOversize
	}

	builder := txbuilder.NewBuilder(time.Now())
	if err = builder.AddInput(types.NewCoinbaseInput(arbitrary), &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}

	coinbaseAmount := consensus.BlockSubsidy(b.block.Height)
	if err = builder.AddOutput(types.NewOriginalTxOutput(*consensus.BTMAssetID, coinbaseAmount + feeAmount, script, [][]byte{})); err != nil {
		return nil, err
	}
	//TODO: calculate reward to proposer

	_, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, err
	}

	txData.SerializedSize = uint64(len(byteData))
	tx = &types.Tx{
		TxData: *txData,
		Tx:     types.MapTx(txData),
	}
	return tx, nil
}

func (b *blockBuilder) applyTransactions(txs []*protocol.TxDesc, timeoutStatus uint8) (uint64, error) {
	var feeAmount uint64
	batchTxs := []*protocol.TxDesc{}
	for i := 0; i < len(txs); i++ {
		if batchTxs = append(batchTxs, txs[i]); len(batchTxs) < batchApplyNum && i != len(txs)-1 {
			continue
		}

		results, gasLeft := b.preValidateTxs(batchTxs, b.chain, b.utxoView, b.gasLeft)
		for j, result := range results {
			if result.err != nil {
				log.WithFields(log.Fields{"module": logModule, "error": result.err}).Error("propose block generation: skip tx due to")
				b.chain.GetTxPool().RemoveTransaction(&result.tx.ID)
				continue
			}

			b.block.Transactions = append(b.block.Transactions, result.tx)
			feeAmount += batchTxs[j].Fee
		}

		b.gasLeft = gasLeft
		batchTxs = batchTxs[:0]
		if b.getTimeoutStatus() >= timeoutStatus || len(b.block.Transactions) > softMaxTxNum {
			break
		}
	}
	return feeAmount, nil
}

type validateTxResult struct {
	tx  *types.Tx
	err error
}

func (b *blockBuilder) preValidateTxs(txs []*protocol.TxDesc, chain *protocol.Chain, view *state.UtxoViewpoint, gasLeft int64) ([]*validateTxResult, int64) {
	var results []*validateTxResult
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: chain.BestBlockHeight() + 1}}
	bcTxs := make([]*bc.Tx, len(txs))
	for i, tx := range txs {
		bcTxs[i] = tx.Tx.Tx
	}

	validateResults := validation.ValidateTxs(bcTxs, bcBlock, b.chain.ProgramConverter)
	for i := 0; i < len(validateResults) && gasLeft > 0; i++ {
		tx := txs[i].Tx
		gasStatus := validateResults[i].GetGasState()
		if err := validateResults[i].GetError(); err != nil {
			results = append(results, &validateTxResult{tx: tx, err: err})
			continue
		}

		if err := chain.GetTransactionsUtxo(view, []*bc.Tx{bcTxs[i]}); err != nil {
			results = append(results, &validateTxResult{tx: tx, err: err})
			continue
		}

		if gasLeft-gasStatus.GasUsed < 0 {
			break
		}

		if err := view.ApplyTransaction(bcBlock, bcTxs[i]); err != nil {
			results = append(results, &validateTxResult{tx: tx, err: err})
			continue
		}

		results = append(results, &validateTxResult{tx: tx, err: validateResults[i].GetError()})
		gasLeft -= gasStatus.GasUsed
	}
	return results, gasLeft
}

func (b *blockBuilder) getTimeoutStatus() uint8 {
	if b.timeoutStatus == timeoutCritical {
		return b.timeoutStatus
	}

	select {
	case <-b.criticalTimeoutCh:
		b.timeoutStatus = timeoutCritical
	case <-b.warnTimeoutCh:
		b.timeoutStatus = timeoutWarn
	default:
	}

	return b.timeoutStatus
}
