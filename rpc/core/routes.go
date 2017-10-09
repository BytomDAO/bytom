package core

import (
	rpc "github.com/bytom/rpc/lib/server"
)

// TODO: better system than "unsafe" prefix
var Routes = map[string]*rpc.RPCFunc{
	// subscribe/unsubscribe are reserved for websocket events.
	"net_info":       rpc.NewRPCFunc(NetInfo, ""),
	"getwork":        rpc.NewRPCFunc(GetWork, ""),
	"submitwork":     rpc.NewRPCFunc(SubmitWork, "height"),
	"getBlockHeight": rpc.NewRPCFunc(BlockHeight, ""),
}

func AddUnsafeRoutes() {
	// control API
	Routes["dial_seeds"] = rpc.NewRPCFunc(UnsafeDialSeeds, "seeds")
}
