package core

import (
	ctypes "github.com/bytom/rpc/core/types"
)

func BlockHeight() (*ctypes.ResultBlockchainInfo, error) {
	return &ctypes.ResultBlockchainInfo{LastHeight: blockStore.Height(),}, nil
}
