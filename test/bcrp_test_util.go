package test

import (
	"github.com/bytom/bytom/database"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol"
)

// MockChainWithStore mock chain with store
func MockChainWithStore(store *database.Store) (*protocol.Chain, *database.Store, *protocol.TxPool, error) {
	dispatcher := event.NewDispatcher()
	txPool := protocol.NewTxPool(store, dispatcher)
	chain, err := protocol.NewChain(store, txPool, dispatcher)
	return chain, store, txPool, err
}
