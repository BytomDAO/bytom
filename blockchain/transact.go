package blockchain

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/blockchain/txbuilder"
	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httperror"
	"github.com/bytom/net/http/reqid"
	"github.com/bytom/protocol/bc/legacy"
)

var defaultTxTTL = 5 * time.Minute

func (a *BlockchainReactor) actionDecoder(action string) (func([]byte) (txbuilder.Action, error), bool) {
	var decoder func([]byte) (txbuilder.Action, error)
	switch action {
	case "control_account":
		decoder = a.accounts.DecodeControlAction
	case "control_address":
		decoder = txbuilder.DecodeControlAddressAction
	case "control_program":
		decoder = txbuilder.DecodeControlProgramAction
	case "control_receiver":
		decoder = txbuilder.DecodeControlReceiverAction
	case "issue":
		decoder = a.assets.DecodeIssueAction
	case "retire":
		decoder = txbuilder.DecodeRetireAction
	case "spend_account":
		decoder = a.accounts.DecodeSpendAction
	case "spend_account_unspent_output":
		decoder = a.accounts.DecodeSpendUTXOAction
	case "set_transaction_reference_data":
		decoder = txbuilder.DecodeSetTxRefDataAction
	default:
		return nil, false
	}
	return decoder, true
}

func (a *BlockchainReactor) buildSingle(ctx context.Context, req *BuildRequest) (*txbuilder.Template, error) {
	err := a.filterAliases(ctx, req)
	if err != nil {
		return nil, err
	}
	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for i, act := range req.Actions {
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

	tpl, err := txbuilder.Build(ctx, req.Tx, actions, maxTime)
	if errors.Root(err) == txbuilder.ErrAction {
		// Format each of the inner errors contained in the data.
		var formattedErrs []httperror.Response
		for _, innerErr := range errors.Data(err)["actions"].([]error) {
			resp := errorFormatter.Format(innerErr)
			formattedErrs = append(formattedErrs, resp)
		}
		err = errors.WithData(err, "actions", formattedErrs)
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
func (a *BlockchainReactor) build(ctx context.Context, buildReqs []*BuildRequest) (interface{}, error) {
	responses := make([]interface{}, len(buildReqs))
	var wg sync.WaitGroup
	wg.Add(len(responses))

	for i := 0; i < len(responses); i++ {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			tmpl, err := a.buildSingle(subctx, buildReqs[i])
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = tmpl
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}

func (a *BlockchainReactor) submitSingle(ctx context.Context, tpl *txbuilder.Template, waitUntil string) (interface{}, error) {
	if tpl.Transaction == nil {
		return nil, errors.Wrap(txbuilder.ErrMissingRawTx)
	}

	err := a.finalizeTxWait(ctx, tpl, waitUntil)
	if err != nil {
		return nil, errors.Wrapf(err, "tx %s", tpl.Transaction.ID.String())
	}

	return map[string]string{"id": tpl.Transaction.ID.String()}, nil
}

// finalizeTxWait calls FinalizeTx and then waits for confirmation of
// the transaction.  A nil error return means the transaction is
// confirmed on the blockchain.  ErrRejected means a conflicting tx is
// on the blockchain.  context.DeadlineExceeded means ctx is an
// expiring context that timed out.
func (a *BlockchainReactor) finalizeTxWait(ctx context.Context, txTemplate *txbuilder.Template, waitUntil string) error {
	// Use the current generator height as the lower bound of the block height
	// that the transaction may appear in.
	localHeight := a.chain.Height()
	//generatorHeight := localHeight

	log.WithField("localHeight", localHeight).Info("Starting to finalize transaction")

	err := txbuilder.FinalizeTx(ctx, a.chain, txTemplate.Transaction)
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

func (a *BlockchainReactor) waitForTxInBlock(ctx context.Context, tx *legacy.Tx, height uint64) (uint64, error) {
	log.Printf("waitForTxInBlock function")
	for {
		height++
		select {
		case <-ctx.Done():
			return 0, ctx.Err()

		case <-a.chain.BlockWaiter(height):
			b, err := a.chain.GetBlockByHeight(height)
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
			err = txbuilder.FinalizeTx(ctx, a.chain, tx)
			if err != nil {
				return 0, err
			}

			// TODO(jackson): Do simple rejection checks like checking if
			// the tx's blockchain prevouts still exist in the state tree.
		}
	}
}

type SubmitArg struct {
	Transactions []txbuilder.Template
	Wait         chainjson.Duration
	WaitUntil    string `json:"wait_until"` // values none, confirmed, processed. default: processed
}

// POST /submit-transaction
func (a *BlockchainReactor) submit(ctx context.Context, x SubmitArg) (interface{}, error) {
	// Setup a timeout for the provided wait duration.
	timeout := x.Wait.Duration
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	responses := make([]interface{}, len(x.Transactions))
	var wg sync.WaitGroup
	wg.Add(len(responses))
	for i := range responses {
		go func(i int) {
			subctx := reqid.NewSubContext(ctx, reqid.New())
			defer wg.Done()
			defer batchRecover(subctx, &responses[i])

			tx, err := a.submitSingle(subctx, &x.Transactions[i], x.WaitUntil)
			log.WithFields(log.Fields{"err": err, "tx": tx}).Info("submit single tx")
			if err != nil {
				responses[i] = err
			} else {
				responses[i] = tx
			}
		}(i)
	}

	wg.Wait()
	return responses, nil
}
