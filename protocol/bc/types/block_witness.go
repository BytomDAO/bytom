package types

import (
	"io"

	"github.com/bytom/bytom/encoding/blockchain"
)

// BlockWitness save the consensus node sign
type BlockWitness []byte

func (bw *BlockWitness) readFrom(r *blockchain.Reader) (err error) {
	*bw, err = blockchain.ReadVarstr31(r)
	return err
}

func (bw *BlockWitness) writeTo(w io.Writer) error {
	_, err := blockchain.WriteVarstr31(w, *bw)
	return err
}

func (bw *BlockWitness) Set(data []byte) {
	copy(*bw, data)
}
