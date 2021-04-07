package mock

import (
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/bc/types"
)

type Mempool struct {
	txs []*protocol.TxDesc
}

func newMempool() *Mempool {
	return &Mempool{
		txs: []*protocol.TxDesc{},
	}
}

func (m *Mempool) AddTx(tx *types.Tx) {
	m.txs = append(m.txs, &protocol.TxDesc{Tx: tx})
}

func (m *Mempool) GetTransactions() []*protocol.TxDesc {
	return m.txs
}

func (m *Mempool) IsDust(tx *types.Tx) bool {
	return false
}
