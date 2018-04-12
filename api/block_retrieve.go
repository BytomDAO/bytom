package api

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/consensus/difficulty"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// return best block hash
func (a *API) getBestBlockHash() Response {
	blockHash := map[string]string{"blockHash": a.chain.BestBlockHash().String()}
	return NewSuccessResponse(blockHash)
}

// return block header by hash
func (a *API) getBlockHeaderByHash(strHash string) Response {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return NewErrorResponse(err)
	}
	block, err := a.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return NewErrorResponse(err)
	}

	bcBlock := types.MapBlock(block)
	return NewSuccessResponse(bcBlock.BlockHeader)
}

// BlockTx is the tx struct for getBlock func
type BlockTx struct {
	ID         bc.Hash                  `json:"id"`
	Version    uint64                   `json:"version"`
	Size       uint64                   `json:"size"`
	TimeRange  uint64                   `json:"time_range"`
	Inputs     []*query.AnnotatedInput  `json:"inputs"`
	Outputs    []*query.AnnotatedOutput `json:"outputs"`
	StatusFail bool                     `json:"status_fail"`
}

// GetBlockReq is used to handle getBlock req
type GetBlockReq struct {
	BlockHeight uint64             `json:"block_height"`
	BlockHash   chainjson.HexBytes `json:"block_hash"`
}

// GetBlockResp is the resp for getBlock api
type GetBlockResp struct {
	Hash                   *bc.Hash   `json:"hash"`
	Size                   uint64     `json:"size"`
	Version                uint64     `json:"version"`
	Height                 uint64     `json:"height"`
	PreviousBlockHash      *bc.Hash   `json:"previous_block_hash"`
	Timestamp              uint64     `json:"timestamp"`
	Nonce                  uint64     `json:"nonce"`
	Bits                   uint64     `json:"bits"`
	Difficulty             string     `json:"difficulty"`
	TransactionsMerkleRoot *bc.Hash   `json:"transaction_merkle_root"`
	TransactionStatusHash  *bc.Hash   `json:"transaction_status_hash"`
	Transactions           []*BlockTx `json:"transactions"`
}

// return block by hash
func (a *API) getBlock(ins GetBlockReq) Response {
	var err error
	block := &types.Block{}
	if len(ins.BlockHash) == 32 {
		b32 := [32]byte{}
		copy(b32[:], ins.BlockHash)
		hash := bc.NewHash(b32)
		block, err = a.chain.GetBlockByHash(&hash)
	} else {
		block, err = a.chain.GetBlockByHeight(ins.BlockHeight)
	}
	if err != nil {
		return NewErrorResponse(err)
	}

	blockHash := block.Hash()
	txStatus, err := a.chain.GetTransactionStatus(&blockHash)
	rawBlock, err := block.MarshalText()
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &GetBlockResp{
		Hash:                   &blockHash,
		Size:                   uint64(len(rawBlock)),
		Version:                block.Version,
		Height:                 block.Height,
		PreviousBlockHash:      &block.PreviousBlockHash,
		Timestamp:              block.Timestamp,
		Nonce:                  block.Nonce,
		Bits:                   block.Bits,
		Difficulty:             difficulty.CompactToBig(block.Bits).String(),
		TransactionsMerkleRoot: &block.TransactionsMerkleRoot,
		TransactionStatusHash:  &block.TransactionStatusHash,
		Transactions:           []*BlockTx{},
	}

	for i, orig := range block.Transactions {
		tx := &BlockTx{
			ID:        orig.ID,
			Version:   orig.Version,
			Size:      orig.SerializedSize,
			TimeRange: orig.TimeRange,
			Inputs:    []*query.AnnotatedInput{},
			Outputs:   []*query.AnnotatedOutput{},
		}
		tx.StatusFail, err = txStatus.GetStatus(i)
		if err != nil {
			NewSuccessResponse(resp)
		}

		for i := range orig.Inputs {
			tx.Inputs = append(tx.Inputs, a.wallet.BuildAnnotatedInput(orig, uint32(i)))
		}
		for i := range orig.Outputs {
			tx.Outputs = append(tx.Outputs, a.wallet.BuildAnnotatedOutput(orig, i))
		}
		resp.Transactions = append(resp.Transactions, tx)
	}
	return NewSuccessResponse(resp)
}

// return block transactions count by hash
func (a *API) getBlockTransactionsCountByHash(strHash string) Response {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return NewErrorResponse(err)
	}

	legacyBlock, err := a.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return NewErrorResponse(err)
	}

	count := map[string]int{"count": len(legacyBlock.Transactions)}
	return NewSuccessResponse(count)
}

// return block transactions count by height
func (a *API) getBlockTransactionsCountByHeight(height uint64) Response {
	legacyBlock, err := a.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return NewErrorResponse(err)
	}

	count := map[string]int{"count": len(legacyBlock.Transactions)}
	return NewSuccessResponse(count)
}

// return current block count
func (a *API) getBlockCount() Response {
	blockHeight := map[string]uint64{"block_count": a.chain.BestBlockHeight()}
	return NewSuccessResponse(blockHeight)
}
