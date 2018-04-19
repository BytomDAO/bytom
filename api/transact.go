package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/errors"
	"github.com/bytom/math/checked"
	"github.com/bytom/net/http/reqid"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
)

var defaultTxTTL = 5 * time.Minute

func (a *API) actionDecoder(action string) (func([]byte) (txbuilder.Action, error), bool) {
	var decoder func([]byte) (txbuilder.Action, error)
	switch action {
	case "control_address":
		decoder = txbuilder.DecodeControlAddressAction
	case "control_program":
		decoder = txbuilder.DecodeControlProgramAction
	case "control_receiver":
		decoder = txbuilder.DecodeControlReceiverAction
	case "issue":
		decoder = a.wallet.AssetReg.DecodeIssueAction
	case "retire":
		decoder = txbuilder.DecodeRetireAction
	case "spend_account":
		decoder = a.wallet.AccountMgr.DecodeSpendAction
	case "spend_account_unspent_output":
		decoder = a.wallet.AccountMgr.DecodeSpendUTXOAction
	default:
		return nil, false
	}
	return decoder, true
}

func mergeActions(req *BuildRequest) []map[string]interface{} {
	var actions []map[string]interface{}
	actionMap := make(map[string]map[string]interface{})

	for _, m := range req.Actions {
		if actionType := m["type"].(string); actionType != "spend_account" {
			actions = append(actions, m)
			continue
		}

		actionKey := m["asset_id"].(string) + m["account_id"].(string)
		amountNumber := m["amount"].(json.Number)
		amount, _ := amountNumber.Int64()

		if tmpM, ok := actionMap[actionKey]; ok {
			tmpNumber, _ := tmpM["amount"].(json.Number)
			tmpAmount, _ := tmpNumber.Int64()
			tmpM["amount"] = json.Number(fmt.Sprintf("%v", tmpAmount+amount))
		} else {
			actionMap[actionKey] = m
			actions = append(actions, m)
		}
	}

	return actions
}

func onlyHaveSpendActions(req *BuildRequest) bool {
	count := 0
	for _, m := range req.Actions {
		if actionType := m["type"].(string); strings.HasPrefix(actionType, "spend") {
			count++
		}
	}

	return count == len(req.Actions)
}

func (a *API) buildSingle(ctx context.Context, req *BuildRequest) (*txbuilder.Template, error) {
	err := a.filterAliases(ctx, req)
	if err != nil {
		return nil, err
	}

	if onlyHaveSpendActions(req) {
		return nil, errors.New("transaction only contain spend actions, didn't have output actions")
	}

	reqActions := mergeActions(req)
	actions := make([]txbuilder.Action, 0, len(reqActions))
	for i, act := range reqActions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "no action type provided on action %d", i)
		}
		decoder, ok := a.actionDecoder(typ)
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "unknown action type %q on action %d", typ, i)
		}

		// Remarshal to JSON, the action may have been modified when we
		// filtered aliases.
		b, err := json.Marshal(act)
		if err != nil {
			return nil, err
		}
		action, err := decoder(b)
		if err != nil {
			return nil, errors.WithDetailf(errBadAction, "%s on action %d", err.Error(), i)
		}
		actions = append(actions, action)
	}

	ttl := req.TTL.Duration
	if ttl == 0 {
		ttl = defaultTxTTL
	}
	maxTime := time.Now().Add(ttl)

	tpl, err := txbuilder.Build(ctx, req.Tx, actions, maxTime, req.TimeRange)
	if errors.Root(err) == txbuilder.ErrAction {
		// append each of the inner errors contained in the data.
		var Errs string
		for _, innerErr := range errors.Data(err)["actions"].([]error) {
			Errs = Errs + "<" + innerErr.Error() + ">"
		}
		err = errors.New(err.Error() + "-" + Errs)
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

	log.WithField("tx_id", ins.Tx.ID).Info("submit single tx")
	return NewSuccessResponse(&submitTxResp{TxID: &ins.Tx.ID})
}

// EstimateTxGasResp estimate transaction consumed gas
type EstimateTxGasResp struct {
	TotalGas   int64 `json:"total_gas"`
	StorageGas int64 `json:"storage_gas"`
	VMGas      int64 `json:"vm_gas"`
}

// POST /estimate-transaction-gas
func (a *API) estimateTxGas(ctx context.Context, in struct {
	TxTemplate txbuilder.Template `json:"transaction_template"`
}) Response {
	// base tx size and not include sign
	data, err := in.TxTemplate.Transaction.TxData.MarshalText()
	if err != nil {
		return NewErrorResponse(err)
	}
	baseTxSize := int64(len(data))

	// extra tx size for sign witness parts
	baseWitnessSize := int64(300)
	lenSignInst := int64(len(in.TxTemplate.SigningInstructions))
	signSize := baseWitnessSize * lenSignInst

	// total gas for tx storage
	totalTxSizeGas, ok := checked.MulInt64(baseTxSize+signSize, consensus.StorageGasRate)
	if !ok {
		return NewErrorResponse(errors.New("calculate txsize gas got a math error"))
	}

	// consume gas for run VM
	totalP2WPKHGas := int64(0)
	totalP2WSHGas := int64(0)
	baseP2WPKHGas := int64(1419)
	baseP2WSHGas := int64(2499)

	for _, inpID := range in.TxTemplate.Transaction.Tx.InputIDs {
		sp, err := in.TxTemplate.Transaction.Spend(inpID)
		if err != nil {
			continue
		}

		resOut, ok := in.TxTemplate.Transaction.Entries[*sp.SpentOutputId].(*bc.Output)
		if !ok {
			continue
		}

		if segwit.IsP2WPKHScript(resOut.ControlProgram.Code) {
			totalP2WPKHGas += baseP2WPKHGas
		} else if segwit.IsP2WPKHScript(resOut.ControlProgram.Code) {
			totalP2WSHGas += baseP2WSHGas
		}
	}

	// total estimate gas
	totalGas := totalTxSizeGas + totalP2WPKHGas + totalP2WSHGas

	txGasResp := &EstimateTxGasResp{
		TotalGas:   totalGas,
		StorageGas: totalTxSizeGas,
		VMGas:      totalP2WPKHGas + totalP2WSHGas,
	}

	return NewSuccessResponse(txGasResp)
}
