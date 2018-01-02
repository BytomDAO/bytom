package blockchain

import (
	"context"

	"github.com/bytom/blockchain/account"
	"github.com/bytom/blockchain/query"
	"github.com/bytom/blockchain/query/filter"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httperror"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/protocol"
)

func isTemporary(info httperror.Info, err error) bool {
	switch info.ChainCode {
	case "BTM000": // internal server error
		return true
	case "BTM001": // request timed out
		return true
	case "BTM761": // outputs currently reserved
		return true
	case "BTM706": // 1 or more action errors
		errs := errors.Data(err)["actions"].([]httperror.Response)
		temp := true
		for _, actionErr := range errs {
			temp = temp && isTemporary(actionErr.Info, nil)
		}
		return temp
	default:
		return false
	}
}

// Map error values to standard bytom error codes. Missing entries
// will map to internalErrInfo.
//
// TODO(jackson): Share one error table across Chain
// products/services so that errors are consistent.
var errorFormatter = httperror.Formatter{
	Default:     httperror.Info{500, "BTM000", "Bytom API Error"},
	IsTemporary: isTemporary,
	Errors: map[error]httperror.Info{
		// General error namespace (0xx)
		context.DeadlineExceeded: {408, "BTM001", "Request timed out"},
		httpjson.ErrBadRequest:   {400, "BTM003", "Invalid request body"},
		txbuilder.ErrMissingFields: {400, "BTM010", "One or more fields are missing"},
		rpc.ErrWrongNetwork:          {502, "BTM104", "A peer core is operating on a different blockchain network"},
		protocol.ErrTheDistantFuture: {400, "BTM105", "Requested height is too far ahead"},

		// Signers error namespace (2xx)
		signers.ErrBadQuorum: {400, "BTM200", "Quorum must be greater than 1 and less than or equal to the length of xpubs"},
		signers.ErrBadXPub:   {400, "BTM201", "Invalid xpub format"},
		signers.ErrNoXPubs:   {400, "BTM202", "At least one xpub is required"},
		signers.ErrBadType:   {400, "BTM203", "Retrieved type does not match expected type"},
		signers.ErrDupeXPub:  {400, "BTM204", "Root XPubs cannot contain the same key more than once"},

		// Query error namespace (6xx)
		query.ErrBadAfter:               {400, "BTM600", "Malformed pagination parameter `after`"},
		query.ErrParameterCountMismatch: {400, "BTM601", "Incorrect number of parameters to filter"},
		filter.ErrBadFilter:             {400, "BTM602", "Malformed query filter"},

		// Transaction error namespace (7xx)
		// Build error namespace (70x)
		txbuilder.ErrBadRefData: {400, "BTM700", "Reference data does not match previous transaction's reference data"},
		txbuilder.ErrBadAmount:  {400, "BTM704", "Invalid asset amount"},
		txbuilder.ErrBlankCheck: {400, "BTM705", "Unsafe transaction: leaves assets to be taken without requiring payment"},
		txbuilder.ErrAction:     {400, "BTM706", "One or more actions had an error: see attached data"},

		// Submit error namespace (73x)
		txbuilder.ErrMissingRawTx:          {400, "BTM730", "Missing raw transaction"},
		txbuilder.ErrBadInstructionCount:   {400, "BTM731", "Too many signing instructions in template for transaction"},
		txbuilder.ErrBadTxInputIdx:         {400, "BTM732", "Invalid transaction input index"},
		txbuilder.ErrBadWitnessComponent:   {400, "BTM733", "Invalid witness component"},
		txbuilder.ErrRejected:              {400, "BTM735", "Transaction rejected"},
		txbuilder.ErrNoTxSighashCommitment: {400, "BTM736", "Transaction is not final, additional actions still allowed"},
		txbuilder.ErrTxSignatureFailure:    {400, "BTM737", "Transaction signature missing, client may be missing signature key"},
		txbuilder.ErrNoTxSighashAttempt:    {400, "BTM738", "Transaction signature was not attempted"},

		// account action error namespace (76x)
		account.ErrInsufficient: {400, "BTM760", "Insufficient funds for tx"},
		account.ErrReserved:     {400, "BTM761", "Some outputs are reserved; try again"},

	},
}
