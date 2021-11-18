package types

import (
	"io"

	"github.com/bytom/bytom/crypto/sm2"
	"github.com/bytom/bytom/encoding/blockchain"
)

// BlockWitness save the consensus node sign
type BlockWitness []byte

// Set write the sign data to BlockWitness
func (bw *BlockWitness) Set(data []byte) {
	witness := make([]byte, sm2.SignatureSize)
	copy(witness, data)
	*bw = witness
}

func (bw *BlockWitness) readFrom(r *blockchain.Reader) (err error) {
	*bw, err = blockchain.ReadVarstr31(r)
	return err
}

func (bw *BlockWitness) writeTo(w io.Writer) error {
	_, err := blockchain.WriteVarstr31(w, *bw)
	return err
}
