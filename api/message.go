package api

import (
	"context"
	"encoding/hex"
	"strings"

	"github.com/bytom/blockchain/signers"
	"github.com/bytom/common"
	"github.com/bytom/consensus"
	"github.com/bytom/crypto"
	"github.com/bytom/crypto/ed25519"
	"github.com/bytom/crypto/ed25519/chainkd"
)

// SignMsgResp is response for sign message
type SignMsgResp struct {
	Signature   string       `json:"signature"`
	DerivedXPub chainkd.XPub `json:"derived_xpub"`
}

func (a *API) signMessage(ctx context.Context, ins struct {
	Address  string `json:"address"`
	Message  string `json:"message"`
	Password string `json:"password"`
}) Response {
	cp, err := a.wallet.AccountMgr.GetProgramByAddress(ins.Address)
	if err != nil {
		return NewErrorResponse(err)
	}

	account, err := a.wallet.AccountMgr.GetAccountByProgram(cp)
	if err != nil {
		return NewErrorResponse(err)
	}

	path := signers.Path(account.Signer, signers.AccountKeySpace, cp.KeyIndex)
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)

	sig, err := a.wallet.Hsm.XSign(account.XPubs[0], path, []byte(ins.Message), ins.Password)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(SignMsgResp{
		Signature:   hex.EncodeToString(sig),
		DerivedXPub: derivedXPubs[0],
	})
}

// VerifyMsgResp is response for verify message
type VerifyMsgResp struct {
	VerifyResult bool `json:"result"`
}

func (a *API) verifyMessage(ctx context.Context, ins struct {
	Address     string       `json:"address"`
	DerivedXPub chainkd.XPub `json:"derived_xpub"`
	Message     string       `json:"message"`
	Signature   string       `json:"signature"`
}) Response {
	sig, err := hex.DecodeString(ins.Signature)
	if err != nil {
		return NewErrorResponse(err)
	}

	derivedPK := ins.DerivedXPub.PublicKey()
	pubHash := crypto.Ripemd160(derivedPK)
	addressPubHash, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
	if err != nil {
		return NewErrorResponse(err)
	}

	address := addressPubHash.EncodeAddress()
	if address != strings.TrimSpace(ins.Address) {
		return NewSuccessResponse(VerifyMsgResp{VerifyResult: false})
	}

	if ed25519.Verify(ins.DerivedXPub.PublicKey(), []byte(ins.Message), sig) {
		return NewSuccessResponse(VerifyMsgResp{VerifyResult: true})
	}
	return NewSuccessResponse(VerifyMsgResp{VerifyResult: false})
}
