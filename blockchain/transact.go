package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/contract"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/reqid"
	"github.com/bytom/protocol/bc/legacy"
)

var defaultTxTTL = 5 * time.Minute

func (bcr *BlockchainReactor) actionDecoder(action string) (func([]byte) (txbuilder.Action, error), bool) {
	var decoder func([]byte) (txbuilder.Action, error)
	switch action {
	case "control_account":
		decoder = bcr.accounts.DecodeControlAction
	case "control_address":
		decoder = txbuilder.DecodeControlAddressAction
	case "control_program":
		decoder = txbuilder.DecodeControlProgramAction
	case "control_receiver":
		decoder = txbuilder.DecodeControlReceiverAction
	case "issue":
		decoder = bcr.assets.DecodeIssueAction
	case "retire":
		decoder = txbuilder.DecodeRetireAction
	case "spend_account":
		decoder = bcr.accounts.DecodeSpendAction
	case "spend_account_unspent_output":
		decoder = bcr.accounts.DecodeSpendUTXOAction
	default:
		return nil, false
	}
	return decoder, true
}

func mergeActions(req *BuildRequest) []map[string]interface{} {
	actions := make([]map[string]interface{}, 0)
	actionMap := make(map[string]map[string]interface{})

	for _, m := range req.Actions {
		if actionType := m["type"].(string); actionType != "spend_account" {
			actions = append(actions, m)
			continue
		}

		actionKey := m["asset_id"].(string) + m["account_id"].(string)

		var amount int64
		if reflect.TypeOf(m["amount"]).Kind().String() == "float64" {
			amount = int64(m["amount"].(float64))
		} else {
			amountStr := fmt.Sprintf("%v", m["amount"])
			amount, _ = strconv.ParseInt(amountStr, 10, 64)
		}

		if tmpM, ok := actionMap[actionKey]; ok {
			var tmpAmount int64
			if reflect.TypeOf(tmpM["amount"]).Kind().String() == "float64" {
				tmpAmount = int64(tmpM["amount"].(float64))
			} else {
				tmpAmountStr := fmt.Sprintf("%v", tmpM["amount"])
				tmpAmount, _ = strconv.ParseInt(tmpAmountStr, 10, 64)
			}
			tmpM["amount"] = tmpAmount + amount
		} else {
			actionMap[actionKey] = m
			actions = append(actions, m)
		}
	}

	return actions
}

func (bcr *BlockchainReactor) buildSingle(ctx context.Context, req *BuildRequest) (*txbuilder.Template, error) {
	err := bcr.filterAliases(ctx, req)
	if err != nil {
		return nil, err
	}

	reqActions := mergeActions(req)
	actions := make([]txbuilder.Action, 0, len(reqActions))
	for i, act := range reqActions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(errBadActionType, "no action type provided on action %d", i)
		}
		decoder, ok := bcr.actionDecoder(typ)
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

	tpl, err := txbuilder.Build(ctx, req.Tx, actions, maxTime)
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
func (bcr *BlockchainReactor) build(ctx context.Context, buildReqs *BuildRequest) Response {
	subctx := reqid.NewSubContext(ctx, reqid.New())

	tmpl, err := bcr.buildSingle(subctx, buildReqs)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tmpl)
}

// POST /lock-contract-transaction
func (bcr *BlockchainReactor) lockContractTX(ctx context.Context, buildReqs *BuildRequest) Response {
	subctx := reqid.NewSubContext(ctx, reqid.New())

	tmpl, err := bcr.buildSingle(subctx, buildReqs)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tmpl)
}

// POST /unlock-contract-transaction
func (bcr *BlockchainReactor) unlockContractTX(ctx context.Context, req *contract.ContractReq) Response {
	act, err := req.ContractDecoder()
	if err != nil {
		return NewErrorResponse(err)
	}

	buildReqStr, err := act.Build()
	if err != nil {
		return NewErrorResponse(err)
	}

	var buildReq BuildRequest
	if err := json.Unmarshal([]byte(buildReqStr), &buildReq); err != nil {
		return NewErrorResponse(err)
	}

	tmpl, err := bcr.buildSingle(ctx, &buildReq)
	if err != nil {
		return NewErrorResponse(err)
	}

	if err = act.AddArgs(tmpl); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tmpl)
}

func (bcr *BlockchainReactor) submitSingle(ctx context.Context, tpl *txbuilder.Template) (map[string]string, error) {
	if tpl.Transaction == nil {
		return nil, errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	if err := txbuilder.FinalizeTx(ctx, bcr.chain, tpl.Transaction); err != nil {
		return nil, errors.Wrapf(err, "tx %s", tpl.Transaction.ID.String())
	}

	return map[string]string{"txid": tpl.Transaction.ID.String()}, nil
}

// finalizeTxWait calls FinalizeTx and then waits for confirmation of
// the transaction.  A nil error return means the transaction is
// confirmed on the blockchain.  ErrRejected means a conflicting tx is
// on the blockchain.  context.DeadlineExceeded means ctx is an
// expiring context that timed out.
func (bcr *BlockchainReactor) finalizeTxWait(ctx context.Context, txTemplate *txbuilder.Template, waitUntil string) error {
	// Use the current generator height as the lower bound of the block height
	// that the transaction may appear in.
	localHeight := bcr.chain.Height()
	//generatorHeight := localHeight

	log.WithField("localHeight", localHeight).Info("Starting to finalize transaction")

	err := txbuilder.FinalizeTx(ctx, bcr.chain, txTemplate.Transaction)
	if err != nil {
		return err
	}
	if waitUntil == "none" {
		return nil
	}

	//TODO:complete finalizeTxWait
	//height, err := a.waitForTxInBlock(ctx, txTemplate.Transaction, generatorHeight)
	if err != nil {
		return err
	}
	if waitUntil == "confirmed" {
		return nil
	}

	return nil
}

func (bcr *BlockchainReactor) waitForTxInBlock(ctx context.Context, tx *legacy.Tx, height uint64) (uint64, error) {
	log.Printf("waitForTxInBlock function")
	for {
		height++
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		case <-bcr.chain.BlockWaiter(height):
			b, err := bcr.chain.GetBlockByHeight(height)
			if err != nil {
				return 0, errors.Wrap(err, "getting block that just landed")
			}
			for _, confirmed := range b.Transactions {
				if confirmed.ID == tx.ID {
					// confirmed
					return height, nil
				}
			}

			// might still be in pool or might be rejected; we can't
			// tell definitively until its max time elapses.
			// Re-insert into the pool in case it was dropped.
			err = txbuilder.FinalizeTx(ctx, bcr.chain, tx)
			if err != nil {
				return 0, err
			}

			// TODO(jackson): Do simple rejection checks like checking if
			// the tx's blockchain prevouts still exist in the state tree.
		}
	}
}

// POST /submit-transaction
func (bcr *BlockchainReactor) submit(ctx context.Context, tpl *txbuilder.Template) Response {
	txID, err := bcr.submitSingle(nil, tpl)
	if err != nil {
		log.WithField("err", err).Error("submit single tx")
		return NewErrorResponse(err)
	}

	log.WithField("txid", txID["txid"]).Info("submit single tx")
	return NewSuccessResponse(txID)
}

// POST /sign-submit-transaction
func (bcr *BlockchainReactor) signSubmit(ctx context.Context, x struct {
	Password []string           `json:"password"`
	Txs      txbuilder.Template `json:"transaction"`
}) Response {
	if err := txbuilder.Sign(ctx, &x.Txs, nil, x.Password[0], bcr.pseudohsmSignTemplate); err != nil {
		log.WithField("build err", err).Error("fail on sign transaction.")
		return NewErrorResponse(err)
	}
	log.Info("Sign Transaction complete.")

	txID, err := bcr.submitSingle(nil, &x.Txs)
	if err != nil {
		log.WithField("err", err).Error("submit single tx")
		return NewErrorResponse(err)
	}

	log.WithField("txid", txID["txid"]).Info("submit single tx")
	return NewSuccessResponse(txID)
}
