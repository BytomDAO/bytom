package validation

import (
	"time"

	"github.com/bytom/consensus"
	"github.com/bytom/consensus/difficulty"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/state"
)

var (
	errBadTimestamp          = errors.New("block timestamp is not in the vaild range")
	errBadBits               = errors.New("block bits is invaild")
	errMismatchedBlock       = errors.New("mismatched block")
	errMismatchedMerkleRoot  = errors.New("mismatched merkle root")
	errMisorderedBlockHeight = errors.New("misordered block height")
	errOverBlockLimit        = errors.New("block's gas is over the limit")
	errWorkProof             = errors.New("invalid difficulty proof of work")
	errVersionRegression     = errors.New("version regression")
)

func checkBlockTime(b *bc.Block, parent *state.BlockNode) error {
	if b.Timestamp > uint64(time.Now().Unix())+consensus.MaxTimeOffsetSeconds {
		return errBadTimestamp
	}

	if b.Timestamp <= parent.CalcPastMedianTime() {
		return errBadTimestamp
	}
	return nil
}

func checkCoinbaseAmount(b *bc.Block, amount uint64) error {
	if len(b.Transactions) == 0 {
		return errors.Wrap(errWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	output, err := tx.Output(*tx.TxHeader.ResultIds[0])
	if err != nil {
		return err
	}

	if output.Source.Value.Amount != amount {
		return errors.Wrap(errWrongCoinbaseTransaction, "dismatch output amount")
	}
	return nil
}

// ValidateBlockHeader check the block's header
func ValidateBlockHeader(b *bc.Block, parent *state.BlockNode) error {
	if b.Version < parent.Version {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", parent.Version, b.Version)
	}
	if b.Height != parent.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", parent.Height, b.Height)
	}
	if b.Bits != parent.CalcNextBits() {
		return errBadBits
	}
	if parent.Hash != *b.PreviousBlockId {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", parent.Hash.Bytes(), b.PreviousBlockId.Bytes())
	}
	if err := checkBlockTime(b, parent); err != nil {
		return err
	}
	if !difficulty.CheckProofOfWork(&b.ID, parent.CalcNextSeed(), b.BlockHeader.Bits) {
		return errWorkProof
	}
	return nil
}

// ValidateBlock validates a block and the transactions within.
func ValidateBlock(b *bc.Block, parent *state.BlockNode) error {
	if err := ValidateBlockHeader(b, parent); err != nil {
		return err
	}

	blockGasSum := uint64(0)
	coinbaseAmount := consensus.BlockSubsidy(b.BlockHeader.Height)
	b.TransactionStatus = bc.NewTransactionStatus()

	for i, tx := range b.Transactions {
		gasStatus, err := ValidateTx(tx, b)
		if !gasStatus.GasVaild {
			return errors.Wrapf(err, "validate of transaction %d of %d", i, len(b.Transactions))
		}

		b.TransactionStatus.SetStatus(i, err != nil)
		coinbaseAmount += gasStatus.BTMValue
		if blockGasSum += uint64(gasStatus.GasUsed); blockGasSum > consensus.MaxBlockGas {
			return errOverBlockLimit
		}
	}

	if err := checkCoinbaseAmount(b, coinbaseAmount); err != nil {
		return err
	}

	txMerkleRoot, err := bc.TxMerkleRoot(b.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction id merkle root")
	}
	if txMerkleRoot != *b.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction id merkle root")
	}

	txStatusHash, err := bc.TxStatusMerkleRoot(b.TransactionStatus.VerifyStatus)
	if err != nil {
		return errors.Wrap(err, "computing transaction status merkle root")
	}
	if txStatusHash != *b.TransactionStatusHash {
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction status merkle root")
	}
	return nil
}
