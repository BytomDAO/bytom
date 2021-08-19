package api

import (
	"gopkg.in/fatih/set.v0"

	"github.com/bytom/bytom/blockchain/query"
	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
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
	ID        bc.Hash                  `json:"id"`
	Version   uint64                   `json:"version"`
	Size      uint64                   `json:"size"`
	TimeRange uint64                   `json:"time_range"`
	Inputs    []*query.AnnotatedInput  `json:"inputs"`
	Outputs   []*query.AnnotatedOutput `json:"outputs"`
	MuxID     bc.Hash                  `json:"mux_id"`
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
	Validator              string     `json:"validator"`
	PreviousBlockHash      *bc.Hash   `json:"previous_block_hash"`
	Timestamp              uint64     `json:"timestamp"`
	TransactionsMerkleRoot *bc.Hash   `json:"transaction_merkle_root"`
	Transactions           []*BlockTx `json:"transactions"`
}

// return block by hash/height
func (a *API) getBlock(ins BlockReq) Response {
	block, err := a.getBlockHelper(ins)
	if err != nil {
		return NewErrorResponse(err)
	}

	blockHash := block.Hash()
	rawBlock, err := block.MarshalText()
	if err != nil {
		return NewErrorResponse(err)
	}

	var validatorPubKey string
	if block.Height > 0 {
		validator, err := a.chain.GetValidator(&block.PreviousBlockHash, block.Timestamp)
		if err != nil {
			return NewErrorResponse(err)
		}

		validatorPubKey = validator.PubKey
	}

	resp := &GetBlockResp{
		Hash:                   &blockHash,
		Size:                   uint64(len(rawBlock)),
		Version:                block.Version,
		Height:                 block.Height,
		Validator:              validatorPubKey,
		PreviousBlockHash:      &block.PreviousBlockHash,
		Timestamp:              block.Timestamp,
		TransactionsMerkleRoot: &block.TransactionsMerkleRoot,
		Transactions:           []*BlockTx{},
	}

	for _, orig := range block.Transactions {
		tx := &BlockTx{
			ID:        orig.ID,
			Version:   orig.Version,
			Size:      orig.SerializedSize,
			TimeRange: orig.TimeRange,
			Inputs:    []*query.AnnotatedInput{},
			Outputs:   []*query.AnnotatedOutput{},
		}

		resOutID := orig.ResultIds[0]
		resOut := orig.Entries[*resOutID]
		switch out :=resOut.(type) {
		case *bc.OriginalOutput:
			tx.MuxID = *out.Source.Ref
		case *bc.VoteOutput:
			tx.MuxID = *out.Source.Ref
		case *bc.Retirement:
			tx.MuxID = *out.Source.Ref
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

// GetRawBlockResp is resp struct for getRawBlock API
type GetRawBlockResp struct {
	RawBlock  *types.Block `json:"raw_block"`
	Validator string       `json:"validator"`
}

func (a *API) getRawBlock(ins BlockReq) Response {
	block, err := a.getBlockHelper(ins)
	if err != nil {
		return NewErrorResponse(err)
	}

	var validatorPubKey string
	if block.Height > 0 {
		validator, err := a.chain.GetValidator(&block.PreviousBlockHash, block.Timestamp)
		if err != nil {
			return NewErrorResponse(err)
		}

		validatorPubKey = validator.PubKey
	}

	resp := GetRawBlockResp{
		RawBlock:  block,
		Validator: validatorPubKey,
	}
	return NewSuccessResponse(resp)
}

// GetBlockHeaderResp is resp struct for getBlockHeader API
type GetBlockHeaderResp struct {
	BlockHeader *types.BlockHeader `json:"block_header"`
}

func (a *API) getBlockHeader(ins BlockReq) Response {
	block, err := a.getBlockHelper(ins)
	if err != nil {
		return NewErrorResponse(err)
	}

	resp := &GetBlockHeaderResp{
		BlockHeader: &block.BlockHeader,
	}
	return NewSuccessResponse(resp)
}

func (a *API) getBlockHelper(ins BlockReq) (*types.Block, error) {
	if len(ins.BlockHash) == 32 {
		hash := hexBytesToHash(ins.BlockHash)
		return a.chain.GetBlockByHash(&hash)
	} else {
		return a.chain.GetBlockByHeight(ins.BlockHeight)
	}
}

func hexBytesToHash(hexBytes chainjson.HexBytes) bc.Hash {
	b32 := [32]byte{}
	copy(b32[:], hexBytes)
	return bc.NewHash(b32)
}

// MerkleBlockReq is used to handle getTxOutProof req
type MerkleBlockReq struct {
	TxIDs     []chainjson.HexBytes `json:"tx_ids"`
	BlockHash chainjson.HexBytes   `json:"block_hash"`
}

// GetMerkleBlockResp is resp struct for GetTxOutProof API
type GetMerkleBlockResp struct {
	BlockHeader  types.BlockHeader `json:"block_header"`
	TxHashes     []*bc.Hash        `json:"tx_hashes"`
	Flags        []uint32          `json:"flags"`
	MatchedTxIDs []*bc.Hash        `json:"matched_tx_ids"`
}

func (a *API) getMerkleProof(ins MerkleBlockReq) Response {
	blockReq := BlockReq{BlockHash: ins.BlockHash}
	block, err := a.getBlockHelper(blockReq)
	if err != nil {
		return NewErrorResponse(err)
	}

	matchedTxs := getMatchedTx(block.Transactions, ins.TxIDs)
	var matchedTxIDs []*bc.Hash
	for _, tx := range matchedTxs {
		matchedTxIDs = append(matchedTxIDs, &tx.ID)
	}

	hashes, compactFlags := types.GetTxMerkleTreeProof(block.Transactions, matchedTxs)
	flags := make([]uint32, len(compactFlags))
	for i, flag := range compactFlags {
		flags[i] = uint32(flag)
	}

	resp := &GetMerkleBlockResp{
		BlockHeader:  block.BlockHeader,
		TxHashes:     hashes,
		Flags:        flags,
		MatchedTxIDs: matchedTxIDs,
	}
	return NewSuccessResponse(resp)
}

func getMatchedTx(txs []*types.Tx, filterTxIDs []chainjson.HexBytes) []*types.Tx {
	txIDSet := set.New()
	for _, txID := range filterTxIDs {
		hash := hexBytesToHash(txID)
		txIDSet.Add(hash.String())
	}

	var matchedTxs []*types.Tx
	for _, tx := range txs {
		hashStr := tx.ID.String()
		if txIDSet.Has(hashStr) {
			matchedTxs = append(matchedTxs, tx)
		}
	}
	return matchedTxs
}
