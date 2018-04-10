package protocol

import (
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/validation"
)

var (
	errBadTimestamp             = errors.New("block timestamp is not in the vaild range")
	errBadBits                  = errors.New("block bits is invaild")
	errMismatchedBlock          = errors.New("mismatched block")
	errMismatchedMerkleRoot     = errors.New("mismatched merkle root")
	errMismatchedTxStatus       = errors.New("mismatched transaction status")
	errMismatchedValue          = errors.New("mismatched value")
	errMisorderedBlockHeight    = errors.New("misordered block height")
	errMisorderedBlockTime      = errors.New("misordered block time")
	errNoPrevBlock              = errors.New("no previous block")
	errOverflow                 = errors.New("arithmetic overflow/underflow")
	errOverBlockLimit           = errors.New("block's gas is over the limit")
	errWorkProof                = errors.New("invalid difficulty proof of work")
	errVersionRegression        = errors.New("version regression")
	errWrongBlockSize           = errors.New("block size is too big")
	errWrongTransactionStatus   = errors.New("transaction status is wrong")
	errWrongCoinbaseTransaction = errors.New("wrong coinbase transaction")
	errNotStandardTx            = errors.New("gas transaction is not standard transaction")
)

// ValidateBlock validates a block and the transactions within.
// It does not run the consensus program; for that, see ValidateBlockSig.
func (c *Chain) validateBlock(b *bc.Block) error {
	parent := c.index.GetNode(b.PreviousBlockId)
	if parent == nil {
		return errors.WithDetailf(errNoPrevBlock, "height %d", b.Height)
	}
	if err := validateBlockAgainstPrev(b, parent); err != nil {
		return err
	}

	if !difficulty.CheckProofOfWork(&b.ID, parent.CalcNextSeed(), b.BlockHeader.Bits) {
		return errWorkProof
	}

	b.TransactionStatus = bc.NewTransactionStatus()
	coinbaseValue := consensus.BlockSubsidy(b.BlockHeader.Height)
	gasUsed := uint64(0)
	for i, tx := range b.Transactions {
		gasStatus, err := validation.ValidateTx(tx, b)
		gasOnlyTx := false
		if err != nil {
			if gasStatus == nil || !gasStatus.GasVaild {
				return errors.Wrapf(err, "validity of transaction %d of %d", i, len(b.Transactions))
			}
			gasOnlyTx = true
		}
		b.TransactionStatus.SetStatus(i, gasOnlyTx)
		coinbaseValue += gasStatus.BTMValue
		gasUsed += uint64(gasStatus.GasUsed)
	}

	if gasUsed > consensus.MaxBlockGas {
		return errOverBlockLimit
	}

	// check the coinbase output entry value
	if err := validateCoinbase(b.Transactions[0], coinbaseValue); err != nil {
		return err
	}

	txRoot, err := bc.TxMerkleRoot(b.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction merkle root")
	}

	if txRoot != *b.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "computed %x, current block wants %x", txRoot.Bytes(), b.TransactionsRoot.Bytes())
	}

	txStatusHash, err := bc.TxStatusMerkleRoot(b.TransactionStatus.VerifyStatus)
	if err != nil {
		return err
	}

	if txStatusHash != *b.TransactionStatusHash {
		return errMismatchedTxStatus
	}
	return nil
}

func validateBlockTime(b *bc.Block, parent *BlockNode) error {
	if b.Timestamp > uint64(time.Now().Unix())+consensus.MaxTimeOffsetSeconds {
		return errBadTimestamp
	}

	if b.Timestamp <= parent.CalcPastMedianTime() {
		return errBadTimestamp
	}
	return nil
}

func validateCoinbase(tx *bc.Tx, value uint64) error {
	resultEntry := tx.Entries[*tx.TxHeader.ResultIds[0]]
	output, ok := resultEntry.(*bc.Output)
	if !ok {
		return errors.Wrap(errWrongCoinbaseTransaction, "decode output")
	}

	if output.Source.Value.Amount != value {
		return errors.Wrap(errWrongCoinbaseTransaction, "dismatch output value")
	}

	inputEntry := tx.Entries[tx.InputIDs[0]]
	input, ok := inputEntry.(*bc.Coinbase)
	if !ok {
		return errors.Wrap(errWrongCoinbaseTransaction, "decode input")
	}
	if input.Arbitrary != nil && len(input.Arbitrary) > consensus.CoinbaseArbitrarySizeLimit {
		return errors.Wrap(errWrongCoinbaseTransaction, "coinbase arbitrary is over size")
	}
	return nil
}

func validateBlockAgainstPrev(b *bc.Block, parent *BlockNode) error {
	if b.Version < parent.version {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", parent.version, b.Version)
	}
	if b.Height != parent.height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", parent.height, b.Height)
	}
	if b.Bits != parent.CalcNextBits() {
		return errBadBits
	}
	if parent.Hash != *b.PreviousBlockId {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", parent.Hash.Bytes(), b.PreviousBlockId.Bytes())
	}
	if err := validateBlockTime(b, parent); err != nil {
		return err
	}
	return nil
}
