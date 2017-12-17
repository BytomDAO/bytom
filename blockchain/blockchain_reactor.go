package blockchain

import (
	stdjson "encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/validation"
)

// return network infomation
func (bcr *BlockchainReactor) getNetInfo() []byte {
	type netInfo struct {
		Listening bool `json:"listening"`
		Syncing   bool `json:"syncing"`
		PeerCount int  `json:"peer_count"`
	}
	net := &netInfo{}
	net.Listening = bcr.sw.IsListening()
	net.Syncing = bcr.blockKeeper.IsCaughtUp()
	net.PeerCount = len(bcr.sw.Peers().List())

	return resWrapper(net)
}

// return best block hash
func (bcr *BlockchainReactor) getBestBlockHash() []byte {
	return resWrapper(bcr.chain.BestBlockHash().String())
}

// return block header by hash
func (bcr *BlockchainReactor) getBlockHeaderByHash(strHash string) []byte {
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
	header, err := stdjson.MarshalIndent(bcBlock.BlockHeader, "", " ")
	if err != nil {
		return resWrapper(nil, err)
	}
	return resWrapper(header)
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
func (bcr *BlockchainReactor) getBlockByHash(strHash string) []byte {
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

	ret, err := stdjson.MarshalIndent(res, "", " ")
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(ret)
}

// return block by height
func (bcr *BlockchainReactor) getBlockByHeight(height uint64) []byte {
	legacyBlock, err := bcr.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return DefaultRawResponse
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

	ret, err := stdjson.MarshalIndent(res, "", " ")
	if err != nil {
		return DefaultRawResponse
	}

	return resWrapper(ret)
}

// return block transactions count by hash
func (bcr *BlockchainReactor) getBlockTransactionsCountByHash(strHash string) (int, error) {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return -1, err
	}

	legacyBlock, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return -1, err
	}
	return len(legacyBlock.Transactions), nil
}

// return network  is or not listening
func (bcr *BlockchainReactor) isNetListening() []byte {
	return resWrapper(bcr.sw.IsListening())
}

// return peer count
func (bcr *BlockchainReactor) peerCount() []byte {
	// TODO: use key-value instead of bare value
	return resWrapper(len(bcr.sw.Peers().List()))
}

// return network syncing information
func (bcr *BlockchainReactor) isNetSyncing() []byte {
	return resWrapper(bcr.blockKeeper.IsCaughtUp())
}

// return block transactions count by height
func (bcr *BlockchainReactor) getBlockTransactionsCountByHeight(height uint64) []byte {
	legacyBlock, err := bcr.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return DefaultRawResponse
	}

	return resWrapper(len(legacyBlock.Transactions))
}

// return block height
func (bcr *BlockchainReactor) blockHeight() []byte {
	return resWrapper(bcr.chain.Height())
}

// return is in mining or not
func (bcr *BlockchainReactor) isMining() []byte {
	return resWrapper(bcr.mining.IsMining())
}

// return gasRate
func (bcr *BlockchainReactor) gasRate() []byte {
	return resWrapper(validation.GasRate)
}

// wrapper json for response
func resWrapper(data interface{}, errWrapper ...error) []byte {
	var response Response

	if errWrapper != nil {
		response = Response{Status: FAIL, Msg: errWrapper[0].Error()}
	} else {
		response = Response{Status: SUCCESS, Data: data}
	}

	rawResponse, err := stdjson.Marshal(response)
	if err != nil {
		return DefaultRawResponse
	}

	return rawResponse
}
