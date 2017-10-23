package core

import (
	ctypes "github.com/bytom/rpc/core/types"
)

func BlockHeight() (*ctypes.ResultBlockchainInfo, error) {
	storeStatus := blockStore.GetStoreStatus()
	return &ctypes.ResultBlockchainInfo{LastHeight: storeStatus.Height}, nil
}
