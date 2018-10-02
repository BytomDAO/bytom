package mining

import "github.com/bytom/protocol"

type byTime []*protocol.TxDesc

func (a byTime) Len() int           { return len(a) }
func (a byTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTime) Less(i, j int) bool { return a[i].Added.Before(a[j].Added) }
