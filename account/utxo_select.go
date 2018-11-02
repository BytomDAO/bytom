package account

import (
	"container/list"
	"sort"
)

type selectUTXO func(uk *utxoKeeper, utxos []*UTXO, amount uint64) ([]*UTXO, uint64, uint64)

var utxoSelectStrategyMap = map[string]selectUTXO{
	"default": (*utxoKeeper).defaultSelectUTXO,
}

func (uk *utxoKeeper) defaultSelectUTXO(utxos []*UTXO, amount uint64) ([]*UTXO, uint64, uint64) {
	//sort the utxo by amount, bigger amount in front
	var optAmount, reservedAmount uint64
	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Amount > utxos[j].Amount
	})

	//push all the available utxos into list
	utxoList := list.New()
	for _, u := range utxos {
		if _, ok := uk.reserved[u.OutputID]; ok {
			reservedAmount += u.Amount
			continue
		}
		utxoList.PushBack(u)
	}

	optList := list.New()
	for node := utxoList.Front(); node != nil; node = node.Next() {
		//append utxo if we haven't reached the required amount
		if optAmount < amount {
			optList.PushBack(node.Value)
			optAmount += node.Value.(*UTXO).Amount
			continue
		}

		largestNode := optList.Front()
		replaceList := list.New()
		replaceAmount := optAmount - largestNode.Value.(*UTXO).Amount

		for ; node != nil && replaceList.Len() <= desireUtxoCount-optList.Len(); node = node.Next() {
			replaceList.PushBack(node.Value)
			if replaceAmount += node.Value.(*UTXO).Amount; replaceAmount >= amount {
				optList.Remove(largestNode)
				optList.PushBackList(replaceList)
				optAmount = replaceAmount
				break
			}
		}

		//largestNode remaining the same means that there is nothing to be replaced
		if largestNode == optList.Front() {
			break
		}
	}

	optUtxos := []*UTXO{}
	for e := optList.Front(); e != nil; e = e.Next() {
		optUtxos = append(optUtxos, e.Value.(*UTXO))
	}
	return optUtxos, optAmount, reservedAmount
}
