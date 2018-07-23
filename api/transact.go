package api

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/account"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/errors"
	"github.com/bytom/math/checked"
	"github.com/bytom/net/http/reqid"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

var (
	defaultTxTTL    = 5 * time.Minute
	defaultBaseRate = float64(100000)
)

func (a *API) actionDecoder(action string) (func([]byte) (txbuilder.Action, error), bool) {
	decoders := map[string]func([]byte) (txbuilder.Action, error){
		"control_address":              txbuilder.DecodeControlAddressAction,
		"control_program":              txbuilder.DecodeControlProgramAction,
		"issue":                        a.wallet.AssetReg.DecodeIssueAction,
		"retire":                       txbuilder.DecodeRetireAction,
		"spend_account":                a.wallet.AccountMgr.DecodeSpendAction,
		"spend_account_unspent_output": a.wallet.AccountMgr.DecodeSpendUTXOAction,
	}
	decoder, ok := decoders[action]
	return decoder, ok
}

func onlyHaveInputActions(req *BuildRequest) (bool, error) {
	count := 0
	for i, act := range req.Actions {
		actionType, ok := act["type"].(string)
		if !ok {
			return false, errors.WithDetailf(ErrBadActionType, "no action type provided on action %d", i)
		}

		if strings.HasPrefix(actionType, "spend") || actionType == "issue" {
			count++
		}
	}

	return count == len(req.Actions), nil
}

func (a *API) buildSingle(ctx context.Context, req *BuildRequest) (*txbuilder.Template, error) {
	if err := a.completeMissingIDs(ctx, req); err != nil {
		return nil, err
	}

	if ok, err := onlyHaveInputActions(req); err != nil {
		return nil, err
	} else if ok {
		return nil, errors.WithDetail(ErrBadActionConstruction, "transaction contains only input actions and no output actions")
	}

	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for i, act := range req.Actions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(ErrBadActionType, "no action type provided on action %d", i)
		}
		decoder, ok := a.actionDecoder(typ)
		if !ok {
			return nil, errors.WithDetailf(ErrBadActionType, "unknown action type %q on action %d", typ, i)
		}

		// Remarshal to JSON, the action may have been modified when we
		// filtered aliases.
		b, err := json.Marshal(act)
		if err != nil {
			return nil, err
		}
		action, err := decoder(b)
		if err != nil {
			return nil, errors.WithDetailf(ErrBadAction, "%s on action %d", err.Error(), i)
		}
		actions = append(actions, action)
	}
	actions = account.MergeSpendAction(actions)

	ttl := req.TTL.Duration
	if ttl == 0 {
		ttl = defaultTxTTL
	}
	maxTime := time.Now().Add(ttl)

	tpl, err := txbuilder.Build(ctx, req.Tx, actions, maxTime, req.TimeRange)
	if errors.Root(err) == txbuilder.ErrAction {
		// append each of the inner errors contained in the data.
		var Errs string
		var rootErr error
		for i, innerErr := range errors.Data(err)["actions"].([]error) {
			if i == 0 {
				rootErr = errors.Root(innerErr)
			}
			Errs = Errs + innerErr.Error()
		}
		err = errors.WithDetail(rootErr, Errs)
	}
	if err != nil {
		return nil, err
	}

	// ensure null is never returned for signing instructions
	if tpl.SigningInstructions == nil {
		tpl.SigningInstructions = []*txbuilder.SigningInstruction{}
	}
	return tpl, nil
}

// POST /build-transaction
func (a *API) build(ctx context.Context, buildReqs *BuildRequest) Response {
	subctx := reqid.NewSubContext(ctx, reqid.New())

	tmpl, err := a.buildSingle(subctx, buildReqs)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tmpl)
}

type submitTxResp struct {
	TxID *bc.Hash `json:"tx_id"`
}

// POST /submit-transaction
func (a *API) submit(ctx context.Context, ins struct {
	Tx types.Tx `json:"raw_transaction"`
}) Response {
	if err := txbuilder.FinalizeTx(ctx, a.chain, &ins.Tx); err != nil {
		return NewErrorResponse(err)
	}

	log.WithField("tx_id", ins.Tx.ID.String()).Info("submit single tx")
	return NewSuccessResponse(&submitTxResp{TxID: &ins.Tx.ID})
}

// EstimateTxGasResp estimate transaction consumed gas
type EstimateTxGasResp struct {
	TotalNeu   int64 `json:"total_neu"`
	StorageNeu int64 `json:"storage_neu"`
	VMNeu      int64 `json:"vm_neu"`
}

// EstimateTxGas estimate consumed neu for transaction
func EstimateTxGas(template txbuilder.Template) (*EstimateTxGasResp, error) {
	// base tx size and not include sign
	data, err := template.Transaction.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	baseTxSize := int64(len(data))

	// extra tx size for sign witness parts
	signSize := estimateSignSize(template.SigningInstructions)

	// total gas for tx storage
	totalTxSizeGas, ok := checked.MulInt64(baseTxSize+signSize, consensus.StorageGasRate)
	if !ok {
		return nil, errors.New("calculate txsize gas got a math error")
	}

	// consume gas for run VM
	totalP2WPKHGas := int64(0)
	totalP2WSHGas := int64(0)
	baseP2WPKHGas := int64(1419)

	for pos, inpID := range template.Transaction.Tx.InputIDs {
		sp, err := template.Transaction.Spend(inpID)
		if err != nil {
			continue
		}

		resOut, err := template.Transaction.Output(*sp.SpentOutputId)
		if err != nil {
			continue
		}

		if segwit.IsP2WPKHScript(resOut.ControlProgram.Code) {
			totalP2WPKHGas += baseP2WPKHGas
		} else if segwit.IsP2WSHScript(resOut.ControlProgram.Code) {
			sigInst := template.SigningInstructions[pos]
			totalP2WSHGas += estimateP2WSHGas(sigInst)
		}
	}

	// total estimate gas
	totalGas := totalTxSizeGas + totalP2WPKHGas + totalP2WSHGas

	// rounding totalNeu with base rate 100000
	totalNeu := float64(totalGas*consensus.VMGasRate) / defaultBaseRate
	roundingNeu := math.Ceil(totalNeu)
	estimateNeu := int64(roundingNeu) * int64(defaultBaseRate)

	// TODO add priority

	return &EstimateTxGasResp{
		TotalNeu:   estimateNeu,
		StorageNeu: totalTxSizeGas * consensus.VMGasRate,
		VMNeu:      (totalP2WPKHGas + totalP2WSHGas) * consensus.VMGasRate,
	}, nil
}

// estimate p2wsh gas.
// OP_CHECKMULTISIG consume (984 * a - 72 * b - 63) gas,
// where a represent the num of public keys, and b represent the num of quorum.
func estimateP2WSHGas(sigInst *txbuilder.SigningInstruction) int64 {
	P2WSHGas := int64(0)
	baseP2WSHGas := int64(738)

	for _, witness := range sigInst.WitnessComponents {
		switch t := witness.(type) {
		case *txbuilder.SignatureWitness:
			P2WSHGas += baseP2WSHGas + (984*int64(len(t.Keys)) - 72*int64(t.Quorum) - 63)
		case *txbuilder.RawTxSigWitness:
			P2WSHGas += baseP2WSHGas + (984*int64(len(t.Keys)) - 72*int64(t.Quorum) - 63)
		}
	}
	return P2WSHGas
}

// estimate signature part size.
// if need multi-sign, calculate the size according to the length of keys.
func estimateSignSize(signingInstructions []*txbuilder.SigningInstruction) int64 {
	signSize := int64(0)
	baseWitnessSize := int64(300)

	for _, sigInst := range signingInstructions {
		for _, witness := range sigInst.WitnessComponents {
			switch t := witness.(type) {
			case *txbuilder.SignatureWitness:
				signSize += int64(t.Quorum) * baseWitnessSize
			case *txbuilder.RawTxSigWitness:
				signSize += int64(t.Quorum) * baseWitnessSize
			}
		}
	}
	return signSize
}

// POST /estimate-transaction-gas
func (a *API) estimateTxGas(ctx context.Context, in struct {
	TxTemplate txbuilder.Template `json:"transaction_template"`
}) Response {
	txGasResp, err := EstimateTxGas(in.TxTemplate)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(txGasResp)
}
