package bc

type Block struct {
	*BlockHeader
	ID           Hash
	Transactions []*Tx
}

type Rollback struct {
	Detach []*Hash
	Attach []*Hash
}
