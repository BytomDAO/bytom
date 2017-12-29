package blockchain

import (
	"bytes"
	stdjson "encoding/json"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/rpc"
	ctypes "github.com/bytom/blockchain/rpc/types"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/validation"
)

// return network infomation
func (bcR *BlockchainReactor) getNetInfo() (*ctypes.ResultNetInfo, error) {
	return rpc.NetInfo(bcR.sw)
}

// return best block hash
func (bcr *BlockchainReactor) getBestBlockHash() []byte {
	data := []string{bcr.chain.BestBlockHash().String()}
	return resWrapper(data)
}

// return block header by hash
func (bcr *BlockchainReactor) getBlockHeaderByHash(strHash string) string {
	var buf bytes.Buffer
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
	}
	block, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return ""
	}
	bcBlock := legacy.MapBlock(block)
	header, _ := stdjson.MarshalIndent(bcBlock.BlockHeader, "", "  ")
	buf.WriteString(string(header))
	return buf.String()
}

type TxJSON struct {
	Inputs  []bc.Entry `json:"inputs"`
	Outputs []bc.Entry `json:"outputs"`
}

type GetBlockByHashJSON struct {
	BlockHeader  *bc.BlockHeader `json:"block_header"`
	Transactions []*TxJSON       `json:"transactions"`
}

// return block by hash
func (bcr *BlockchainReactor) getBlockByHash(strHash string) string {
	hash := bc.Hash{}
	if err := hash.UnmarshalText([]byte(strHash)); err != nil {
		log.WithField("error", err).Error("Error occurs when transforming string hash to hash struct")
		return err.Error()
	}

	legacyBlock, err := bcr.chain.GetBlockByHash(&hash)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return err.Error()
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

	ret, err := stdjson.Marshal(res)
	if err != nil {
		return err.Error()
	}
	return string(ret)
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

	ret, err := stdjson.Marshal(res)
	if err != nil {
		return DefaultRawResponse
	}
	data := []string{string(ret)}
	return resWrapper(data)
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
	data := []string{strconv.FormatBool(bcr.sw.IsListening())}
	return resWrapper(data)
}

// return peer count
func (bcr *BlockchainReactor) peerCount() []byte {
	// TODO: use key-value instead of bare value
	data := []string{strconv.FormatInt(int64(len(bcr.sw.Peers().List())), 16)}
	return resWrapper(data)
}

// return network syncing information
func (bcr *BlockchainReactor) isNetSyncing() []byte {
	data := []string{strconv.FormatBool(bcr.blockKeeper.IsCaughtUp())}
	return resWrapper(data)
}

// return block transactions count by height
func (bcr *BlockchainReactor) getBlockTransactionsCountByHeight(height uint64) []byte {
	legacyBlock, err := bcr.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Error("Fail to get block by hash")
		return DefaultRawResponse
	}
	data := []string{strconv.FormatInt(int64(len(legacyBlock.Transactions)), 16)}
	log.Infof("%v", data)
	return resWrapper(data)
}

// return block height
func (bcr *BlockchainReactor) blockHeight() []byte {
	data := []string{strconv.FormatUint(bcr.chain.Height(), 16)}
	return resWrapper(data)
}

// return is in mining or not
func (bcr *BlockchainReactor) isMining() []byte {
	data := []string{strconv.FormatBool(bcr.mining.IsMining())}
	return resWrapper(data)
}

// return gasRate
func (bcr *BlockchainReactor) gasRate() []byte {
	data := []string{strconv.FormatInt(validation.GasRate, 16)}
	return resWrapper(data)
}

// wrapper json for response
func resWrapper(data []string) []byte {
	response := Response{Status: SUCCESS, Data: data}
	rawResponse, err := stdjson.Marshal(response)
	if err != nil {
		return DefaultRawResponse
	}
	return rawResponse
}
