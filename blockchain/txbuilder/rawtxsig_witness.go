package txbuilder

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	chainjson "github.com/bytom/encoding/json"
)

// TODO(bobg): most of the code here is duplicated from
// signature_witness.go and needs refactoring.

// RawTxSigWitness is like SignatureWitness but doesn't involve
// signature programs.
type RawTxSigWitness struct {
	Quorum int                  `json:"quorum"`
	Keys   []keyID              `json:"keys"`
	Sigs   []chainjson.HexBytes `json:"signatures"`
}

func (sw *RawTxSigWitness) sign(ctx context.Context, tpl *Template, index uint32, auth string, signFn SignFunc) error {
	if len(sw.Sigs) < len(sw.Keys) {
		// Each key in sw.Keys may produce a signature in sw.Sigs. Make
		// sure there are enough slots in sw.Sigs and that we preserve any
		// sigs already present.
		newSigs := make([]chainjson.HexBytes, len(sw.Keys))
		copy(newSigs, sw.Sigs)
		sw.Sigs = newSigs
	}
	for i, keyID := range sw.Keys {
		if len(sw.Sigs[i]) > 0 {
			// Already have a signature for this key
			continue
		}
		path := make([][]byte, len(keyID.DerivationPath))
		for i, p := range keyID.DerivationPath {
			path[i] = p
		}
		sigBytes, err := signFn(ctx, keyID.XPub, path, tpl.Hash(index).Byte32(), auth)
		if err != nil {
			log.WithField("err", err).Warningf("computing signature %d", i)
			continue
		}

		// This break is ordered to avoid signing transaction successfully only once for a multiple-sign account
		// that consist of different keys by the same password. Exit immediately when the signature is success,
		// it means that only one signature will be successful in the loop for this multiple-sign account.
		sw.Sigs[i] = sigBytes
		break
	}
	return nil
}

func (sw RawTxSigWitness) materialize(args *[][]byte) error {
	var nsigs int
	for i := 0; i < len(sw.Sigs) && nsigs < sw.Quorum; i++ {
		if len(sw.Sigs[i]) > 0 {
			*args = append(*args, sw.Sigs[i])
			nsigs++
		}
	}
	return nil
}

// MarshalJSON convert struct to json
func (sw RawTxSigWitness) MarshalJSON() ([]byte, error) {
	obj := struct {
		Type   string               `json:"type"`
		Quorum int                  `json:"quorum"`
		Keys   []keyID              `json:"keys"`
		Sigs   []chainjson.HexBytes `json:"signatures"`
	}{
		Type:   "raw_tx_signature",
		Quorum: sw.Quorum,
		Keys:   sw.Keys,
		Sigs:   sw.Sigs,
	}
	return json.Marshal(obj)
}
