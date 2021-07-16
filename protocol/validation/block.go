package validation

import (
	"encoding/hex"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc/types"
	"github.com/bytom/bytom/protocol/state"
)

const logModule = "validation"

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

func checkCoinbaseAmount(b *types.Block, checkpoint *state.Checkpoint) error {
	if len(b.Transactions) == 0 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	for _, output := range tx.Outputs {
		if output.OutputType() != types.OriginalOutputType || *output.AssetId != *consensus.BTMAssetID {
			return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output type or asset")
		}
	}

	if b.Height%consensus.ActiveNetParams.BlocksOfEpoch != 1 {
		if len(tx.Outputs) != 1 || tx.Outputs[0].Amount != 0 {
			return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output number or amount")
		}
		return nil
	}

	return checkoutRewardCoinbase(tx, checkpoint)
}

func checkoutRewardCoinbase(tx *types.Tx, checkpoint *state.Checkpoint) error {
	outputMap := map[string]uint64{}
	for i, output := range tx.Outputs {
		if i == 0 && output.Amount == 0 {
			continue
		}

		outputMap[hex.EncodeToString(output.ControlProgram)] += output.Amount
	}

	if len(outputMap) != len(checkpoint.Rewards) {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output number")
	}

	for cp, amount := range checkpoint.Rewards {
		if outputMap[cp] != amount {
			return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output amount")
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

	if err := checkCoinbaseAmount(b, checkpoint); err != nil {
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
