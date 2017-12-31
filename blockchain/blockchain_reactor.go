package blockchain

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/validation"
)

// return network infomation
func (bcr *BlockchainReactor) getNetInfo() Response {
	type netInfo struct {
		Listening    bool   `json:"listening"`
		Syncing      bool   `json:"syncing"`
		Mining       bool   `json:"mining"`
		PeerCount    int    `json:"peer_count"`
		CurrentBlock uint64 `json:"current_block"`
		HighestBlock uint64 `json:"highest_block"`
	}
	net := &netInfo{}
	net.Listening = bcr.sw.IsListening()
	net.Syncing = bcr.blockKeeper.IsCaughtUp()
	net.Mining = bcr.mining.IsMining()
	net.PeerCount = len(bcr.sw.Peers().List())
	net.CurrentBlock = bcr.blockKeeper.chainHeight
	net.HighestBlock = bcr.blockKeeper.maxPeerHeight

	return resWrapper(net)
}

// return best block hash
func (bcr *BlockchainReactor) getBestBlockHash() Response {
	blockHash := map[string]string{"blockHash": bcr.chain.BestBlockHash().String()}
	return resWrapper(blockHash)
}

// return block header by hash
func (bcr *BlockchainReactor) getBlockHeaderByHash(strHash string) Response {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return resWrapper(nil, err)
	}
	block, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return resWrapper(nil, err)
	}

	bcBlock := legacy.MapBlock(block)
	return resWrapper(bcBlock.BlockHeader)
}

// TxJSON is used for getting block by hash.
type TxJSON struct {
	Inputs  []bc.Entry `json:"inputs"`
	Outputs []bc.Entry `json:"outputs"`
}

// GetBlockByHashJSON is actually a block, include block header and transactions.
type GetBlockByHashJSON struct {
	BlockHeader  *bc.BlockHeader `json:"block_header"`
	Transactions []*TxJSON       `json:"transactions"`
}

// return block by hash
func (bcr *BlockchainReactor) getBlockByHash(strHash string) Response {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return resWrapper(nil, err)
	}

	legacyBlock, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return resWrapper(nil, err)
	}

	bcBlock := legacy.MapBlock(legacyBlock)
	block := &GetBlockByHashJSON{BlockHeader: bcBlock.BlockHeader}
	for _, tx := range bcBlock.Transactions {
		txJSON := &TxJSON{}
		for _, e := range tx.Entries {
			switch e := e.(type) {
			case *bc.Issuance:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Spend:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Retirement:
				txJSON.Outputs = append(txJSON.Outputs, e)
			case *bc.Output:
				txJSON.Outputs = append(txJSON.Outputs, e)
			default:
				continue
			}
		}
		block.Transactions = append(block.Transactions, txJSON)
	}

	return resWrapper(block)
}

// return block by height
func (bcr *BlockchainReactor) getBlockByHeight(height uint64) Response {
	legacyBlock, err := bcr.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return resWrapper(nil, err)
	}

	bcBlock := legacy.MapBlock(legacyBlock)
	res := &GetBlockByHashJSON{BlockHeader: bcBlock.BlockHeader}
	for _, tx := range bcBlock.Transactions {
		txJSON := &TxJSON{}
		for _, e := range tx.Entries {
			switch e := e.(type) {
			case *bc.Issuance:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Spend:
				txJSON.Inputs = append(txJSON.Inputs, e)
			case *bc.Retirement:
				txJSON.Outputs = append(txJSON.Outputs, e)
			case *bc.Output:
				txJSON.Outputs = append(txJSON.Outputs, e)
			default:
				continue
			}
		}
		res.Transactions = append(res.Transactions, txJSON)
	}

	return resWrapper(res)
}

// return block transactions count by hash
func (bcr *BlockchainReactor) getBlockTransactionsCountByHash(strHash string) Response {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return resWrapper(nil, err)
	}

	legacyBlock, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return resWrapper(nil, err)
	}

	count := map[string]int{"count": len(legacyBlock.Transactions)}
	return resWrapper(count)
}

// return block transactions count by height
func (bcr *BlockchainReactor) getBlockTransactionsCountByHeight(height uint64) Response {
	legacyBlock, err := bcr.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return resWrapper(nil, err)
	}

	count := map[string]int{"count": len(legacyBlock.Transactions)}
	return resWrapper(count)
}

// return block height
func (bcr *BlockchainReactor) blockHeight() Response {
	blockHeight := map[string]uint64{"blockHeight": bcr.chain.Height()}
	return resWrapper(blockHeight)
}

// return is in mining or not
func (bcr *BlockchainReactor) isMining() Response {
	IsMining := map[string]bool{"isMining": bcr.mining.IsMining()}
	return resWrapper(IsMining)
}

// return gasRate
func (bcr *BlockchainReactor) gasRate() Response {
	gasrate := map[string]int64{"gasRate": validation.GasRate}
	return resWrapper(gasrate)
}

// wrapper json for response
func resWrapper(data interface{}, errWrapper ...error) Response {
	var response Response

	if errWrapper != nil {
		response = Response{Status: FAIL, Msg: errWrapper[0].Error()}
	} else {
		response = Response{Status: SUCCESS, Data: data}
	}

	return response
}
