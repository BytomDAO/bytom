package blockchain

import (
	"github.com/bytom/blockchain/rpc"

	ctypes "github.com/bytom/blockchain/rpc/types"
)

func (a *BlockchainReactor) getNetInfo() (*ctypes.ResultNetInfo, error) {
	return rpc.NetInfo(a.sw)
}
