package mining

import "github.com/bytom/protocol"

type ByTime []*protocol.TxDesc

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].Added.Unix() < a[j].Added.Unix() }
