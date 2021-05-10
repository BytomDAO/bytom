package wallet

import (
	"github.com/bytom/bytom/account"
	"github.com/bytom/bytom/consensus/segwit"
)

// GetAddressUTXOs return appointed address all asset  unspent UTXOs(outputs)
func (w *Wallet) GetAddressUTXOs(address, id string, unconfirmed, isSmartContract bool) []*account.UTXO {
	prefix := account.UTXOPreFix
	if isSmartContract {
		prefix = account.SUTXOPrefix
	}

	accountUTXOs := []*account.UTXO{}
	if unconfirmed {
		accountUTXOs = w.AccountMgr.ListUnconfirmedUTXO(func(utxo *account.UTXO) bool {
			return segwit.IsP2WScript(utxo.ControlProgram) != isSmartContract && (address == utxo.Address || address == "")
		})
	}

	accountUTXOIter := w.DB.IteratorPrefix([]byte(prefix + id))
	defer accountUTXOIter.Release()

	utxos := getUTXOs(accountUTXOIter, func(utxo *account.UTXO) bool {
		return address == utxo.Address || address == ""
	})
	accountUTXOs = append(accountUTXOs, utxos...)

	return accountUTXOs
}
