package orm

type Utxo struct {
	ID          uint64 `gorm:"primary_key"`
	OutputID    string
	Xpub        string
	VoteAddress string
	VoteNum     uint64
	VoteHeight  uint64
	VetoHeight  uint64
}
