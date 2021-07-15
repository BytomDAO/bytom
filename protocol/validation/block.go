package validation

import (
	"encoding/hex"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

const logModule = "leveldb"

var (
	errBadTimestamp          = errors.New("block timestamp is not in the valid range")
	errBadBits               = errors.New("block bits is invalid")
	errMismatchedBlock       = errors.New("mismatched block")
	errMismatchedMerkleRoot  = errors.New("mismatched merkle root")
	errMisorderedBlockHeight = errors.New("misordered block height")
	errOverBlockLimit        = errors.New("block's gas is over the limit")
	errVersionRegression     = errors.New("version regression")
)

func checkBlockTime(b, parent *types.BlockHeader) error {
	now := uint64(time.Now().UnixNano() / 1e6)
	if b.Timestamp < (parent.Timestamp + consensus.ActiveNetParams.BlockTimeInterval) {
		return errBadTimestamp
	}
	if b.Timestamp > (now + consensus.ActiveNetParams.MaxTimeOffsetMs) {
		return errBadTimestamp
	}

	return nil
}

func checkCoinbaseAmount(b *bc.Block, checkpoint *state.Checkpoint) error {
	if len(b.Transactions) == 0 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	if len(tx.TxHeader.ResultIds) == 0 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "tx header resultIds is empty")
	}

	if b.Height%consensus.ActiveNetParams.BlocksOfEpoch != 1 || b.Height == 1 {
		output, err := tx.OriginalOutput(*tx.TxHeader.ResultIds[0])
		if err != nil {
			return err
		}

		if output.Source.Value.Amount != 0 {
			return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output amount")
		}

		if len(tx.TxHeader.ResultIds) != 1 {
			return errors.Wrap(ErrWrongCoinbaseTransaction, "have more than 1 output")
		}

		return nil
	}

	return checkoutRewardCoinbase(tx, checkpoint)
}

func checkoutRewardCoinbase(tx *bc.Tx, checkpoint *state.Checkpoint) error {
	resultIdLen := len(tx.TxHeader.ResultIds)
	if resultIdLen != len(checkpoint.Rewards) && resultIdLen != len(checkpoint.Rewards)+1 {
		return errors.Wrap(ErrWrongCoinbaseTransaction)
	}

	var startIndex int
	if resultIdLen == len(checkpoint.Rewards)+1 {
		output, err := tx.OriginalOutput(*tx.TxHeader.ResultIds[0])
		if err != nil {
			return err
		}

		if output.Source.Value.Amount != 0 {
			return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output amount")
		}

		startIndex = 1
	}

	rewards := checkpoint.Rewards
	for i := startIndex; i < resultIdLen; i++ {
		output := tx.TxHeader.ResultIds[i]
		out, err := tx.OriginalOutput(*output)
		if err != nil {
			return err
		}

		if rewards[hex.EncodeToString(out.ControlProgram.Code)] != out.Source.Value.Amount {
			return errors.Wrap(ErrWrongCoinbaseTransaction)
		}
	}

	return nil
}

// ValidateBlockHeader check the block's header
func ValidateBlockHeader(b, parent *types.BlockHeader, checkpoint *state.Checkpoint) error {
	if b.Version != 1 {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", parent.Version, b.Version)
	}

	if b.Height != parent.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", parent.Height, b.Height)
	}

	if parentHash := parent.Hash(); parentHash != b.PreviousBlockHash {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", parentHash.Bytes(), b.PreviousBlockHash)
	}

	if err := checkBlockTime(b, parent); err != nil {
		return err
	}

	return verifyBlockSignature(b, checkpoint)
}

func verifyBlockSignature(blockHeader *types.BlockHeader, checkpoint *state.Checkpoint) error {
	validator := checkpoint.GetValidator(blockHeader.Timestamp)
	xPub := chainkd.XPub{}
	pubKey, err := hex.DecodeString(validator.PubKey)
	if err != nil {
		return err
	}

	copy(xPub[:], pubKey)
	if ok := xPub.Verify(blockHeader.Hash().Bytes(), blockHeader.BlockWitness); !ok {
		return errors.New("fail to verify block header signature")
	}

	return nil
}

// ValidateBlock validates a block and the transactions within.
func ValidateBlock(b *types.Block, parent *types.BlockHeader, checkpoint *state.Checkpoint, converter ProgramConverterFunc) error {
	startTime := time.Now()
	if err := ValidateBlockHeader(&b.BlockHeader, parent, checkpoint); err != nil {
		return err
	}

	bcBlock := types.MapBlock(b)
	blockGasSum := uint64(0)
	validateResults := ValidateTxs(bcBlock.Transactions, bcBlock, converter)
	for i, validateResult := range validateResults {
		if validateResult.err != nil {
			return errors.Wrapf(validateResult.err, "validate of transaction %d of %d", i, len(b.Transactions))
		}

		if blockGasSum += uint64(validateResult.gasStatus.GasUsed); blockGasSum > consensus.MaxBlockGas {
			return errOverBlockLimit
		}
	}

	if err := checkCoinbaseAmount(bcBlock, checkpoint); err != nil {
		return err
	}

	txMerkleRoot, err := types.TxMerkleRoot(bcBlock.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction id merkle root")
	}
	if txMerkleRoot != b.TransactionsMerkleRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction id merkle root")
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   b.Height,
		"hash":     bcBlock.ID.String(),
		"duration": time.Since(startTime),
	}).Debug("finish validate block")
	return nil
}
