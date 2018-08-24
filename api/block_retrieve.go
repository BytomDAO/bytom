package api

import (
	"math/big"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/consensus/difficulty"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

// return best block hash
func (a *API) getBestBlockHash() Response {
	blockHash := map[string]string{"block_hash": a.chain.BestBlockHash().String()}
	return NewSuccessResponse(blockHash)
}

// return current block count
func (a *API) getBlockCount() Response {
	blockHeight := map[string]uint64{"block_count": a.chain.BestBlockHeight()}
	return NewSuccessResponse(blockHeight)
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
	MuxID      bc.Hash                  `json:"mux_id"`
}

// BlockReq is used to handle getBlock req
type BlockReq struct {
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

// return block by hash/height
func (a *API) getBlock(ins BlockReq) Response {
	block, err := a.getBlockHelper(ins)
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
		Difficulty:             difficulty.CalcWork(block.Bits).String(),
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
			return NewSuccessResponse(resp)
		}

		resOutID := orig.ResultIds[0]
		resOut, ok := orig.Entries[*resOutID].(*bc.Output)
		if ok {
			tx.MuxID = *resOut.Source.Ref
		} else {
			resRetire, _ := orig.Entries[*resOutID].(*bc.Retirement)
			tx.MuxID = *resRetire.Source.Ref
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

// GetBlockHeaderResp is resp struct for getBlockHeader API
type GetBlockHeaderResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
	Reward      uint64             `json:"reward"`
}

func (a *API) getBlockHeader(ins BlockReq) Response {
	block, err := a.getBlockHelper(ins)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &GetBlockHeaderResp{
		BlockHeader: &block.BlockHeader,
		Reward:      block.Transactions[0].Outputs[0].Amount,
	}
	return NewSuccessResponse(resp)
}

func (a *API) getBlockHelper(ins BlockReq) (*types.Block, error) {
	if len(ins.BlockHash) == 32 {
		b32 := [32]byte{}
		copy(b32[:], ins.BlockHash)
		hash := bc.NewHash(b32)
		return a.chain.GetBlockByHash(&hash)
	} else {
		return a.chain.GetBlockByHeight(ins.BlockHeight)
	}
}

// GetDifficultyResp is resp struct for getDifficulty API
type GetDifficultyResp struct {
	BlockHash   *bc.Hash `json:"hash"`
	BlockHeight uint64   `json:"height"`
	Bits        uint64   `json:"bits"`
	Difficulty  string   `json:"difficulty"`
}

func (a *API) getDifficulty(ins BlockReq) Response {
	block, err := a.getBlockHelper(ins)
	if err != nil {
		return NewErrorResponse(err)
	}

	blockHash := block.Hash()
	resp := &GetDifficultyResp{
		BlockHash:   &blockHash,
		BlockHeight: block.Height,
		Bits:        block.Bits,
		Difficulty:  difficulty.CalcWork(block.Bits).String(),
	}
	return NewSuccessResponse(resp)
}

// getHashRateResp is resp struct for getHashRate API
type getHashRateResp struct {
	BlockHash   *bc.Hash `json:"hash"`
	BlockHeight uint64   `json:"height"`
	HashRate    uint64   `json:"hash_rate"`
}

func (a *API) getHashRate(ins BlockReq) Response {
	if len(ins.BlockHash) != 32 && len(ins.BlockHash) != 0 {
		err := errors.New("Block hash format error.")
		return NewErrorResponse(err)
	}
	if ins.BlockHeight == 0 {
		ins.BlockHeight = a.chain.BestBlockHeight()
	}

	block, err := a.getBlockHelper(ins)
	if err != nil {
		return NewErrorResponse(err)
	}

	preBlock, err := a.chain.GetBlockByHash(&block.PreviousBlockHash)
	if err != nil {
		return NewErrorResponse(err)
	}

	diffTime := block.Timestamp - preBlock.Timestamp
	hashCount := difficulty.CalcWork(block.Bits)
	hashRate := new(big.Int).Div(hashCount, big.NewInt(int64(diffTime)))

	blockHash := block.Hash()
	resp := &getHashRateResp{
		BlockHash:   &blockHash,
		BlockHeight: block.Height,
		HashRate:    hashRate.Uint64(),
	}
	return NewSuccessResponse(resp)
}
