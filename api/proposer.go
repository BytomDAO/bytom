package api

import (
	"context"
	"strconv"

	chainjson "github.com/bytom/bytom/encoding/json"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol/bc/types"
)

type CoinbaseArbitrary struct {
	Arbitrary chainjson.HexBytes `json:"arbitrary"`
}

func (a *API) getCoinbaseArbitrary() Response {
	arbitrary := a.wallet.AccountMgr.GetCoinbaseArbitrary()
	resp := &CoinbaseArbitrary{
		Arbitrary: arbitrary,
	}

	return NewSuccessResponse(resp)
}

// setCoinbaseArbitrary add arbitary data to the reserved coinbase data.
// check function createCoinbaseTx in proposal/proposal.go for detail.
// arbitraryLenLimit is 107 and can be calculated by:
// 	maxHeight := ^uint64(0)
// 	reserved := append([]byte{0x00}, []byte(strconv.FormatUint(maxHeight, 10))...)
// 	arbitraryLenLimit := consensus.CoinbaseArbitrarySizeLimit - len(reserved)
func (a *API) setCoinbaseArbitrary(ctx context.Context, req CoinbaseArbitrary) Response {
	arbitraryLenLimit := 107
	if len(req.Arbitrary) > arbitraryLenLimit {
		err := errors.New("Arbitrary exceeds limit: " + strconv.FormatUint(uint64(arbitraryLenLimit), 10))
		return NewErrorResponse(err)
	}
	a.wallet.AccountMgr.SetCoinbaseArbitrary(req.Arbitrary)
	return a.getCoinbaseArbitrary()
}

// SubmitBlockReq is req struct for submit-block API
type SubmitBlockReq struct {
	Block *types.Block `json:"raw_block"`
}

// submitBlock trys to submit a raw block to the chain
func (a *API) submitBlock(ctx context.Context, req *SubmitBlockReq) Response {
	isOrphan, err := a.chain.ProcessBlock(req.Block)
	if err != nil {
		return NewErrorResponse(err)
	}

	if isOrphan {
		return NewErrorResponse(errors.New("block submitted is orphan"))
	}

	if err = a.eventDispatcher.Post(event.NewProposedBlockEvent{Block: *req.Block}); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(true)
}

// /set-proposing, start or stop proposing
func (a *API) setProposing(in struct {
	IsProposing bool `json:"is_proposing"`
}) Response {
	if in.IsProposing {
		if _, err := a.wallet.AccountMgr.GetProposerAddress(); err != nil {
			return NewErrorResponse(errors.New("proposing address does not exist"))
		}
		return a.startProposing()
	}

	return a.stopProposing()
}

// /is-proposing, return proposing or not
func (a *API) isProposing() Response {
	IsMining := map[string]bool{"is_proposing": a.blockProposer.IsProposing()}
	return NewSuccessResponse(IsMining)
}

func (a *API) startProposing() Response {
	a.blockProposer.Start()
	if !a.blockProposer.IsProposing() {
		return NewErrorResponse(errors.New("Failed to start proposing"))
	}

	return NewSuccessResponse("")
}

func (a *API) stopProposing() Response {
	a.blockProposer.Stop()
	if a.blockProposer.IsProposing() {
		return NewErrorResponse(errors.New("Failed to stop proposing"))
	}

	return NewSuccessResponse("")
}

type proposerResp struct {
	proposerAddress string `json:"proposer_address"`
}

// /get-proposer-address, get proposer address
func (a *API) getProposerAddress() Response {
	proposer, err := a.wallet.AccountMgr.GetProposerAddress()
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(proposerResp{
		proposerAddress: proposer,
	})
}

// /set-proposer-address, set proposer address and control program
func (a *API) setProposerAddress(ctx context.Context, in struct {
	proposerAddress string `json:"proposer_address"`
}) Response {
	proposerAddress, err := a.wallet.AccountMgr.SetProposerAddress(in.proposerAddress)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(proposerResp{
		proposerAddress: proposerAddress,
	})
}
