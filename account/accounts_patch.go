package account

// ListUnconfirmedUTXO list unconfirmed utxos by conditions
func (m *Manager) ListUnconfirmedUTXO(isWant func(*UTXO) bool) []*UTXO {
	var utxos []*UTXO
	for _, utxo := range m.utxoKeeper.ListUnconfirmed() {
		if isWant(utxo) {
			utxos = append(utxos, utxo)
		}
	}
	return utxos
}
