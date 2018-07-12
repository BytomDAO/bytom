package api

import (
	"context"

	"github.com/bytom/account"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/rpc"
	"github.com/bytom/blockchain/signers"
	"github.com/bytom/blockchain/txbuilder"
	"github.com/bytom/errors"
	"github.com/bytom/net/http/httperror"
	"github.com/bytom/net/http/httpjson"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/validation"
	"github.com/bytom/protocol/vm"
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

var respErrFormatter = map[error]httperror.Info{
	// Signers error namespace (2xx)
	signers.ErrBadQuorum: {400, "BTM200", "Quorum must be greater than 1 and less than or equal to the length of xpubs"},
	signers.ErrBadXPub:   {400, "BTM201", "Invalid xpub format"},
	signers.ErrNoXPubs:   {400, "BTM202", "At least one xpub is required"},
	signers.ErrBadType:   {400, "BTM203", "Retrieved type does not match expected type"},
	signers.ErrDupeXPub:  {400, "BTM204", "Root XPubs cannot contain the same key more than once"},

	// Transaction error namespace (7xx)
	// Build transaction error namespace (70x)
	account.ErrInsufficient:    {400, "BTM700", "Funds of account are insufficient"},
	account.ErrImmature:        {400, "BTM701", "Available funds of account are immature"},
	account.ErrReserved:        {400, "BTM702", "Available UTXOs of account have been reserved"},
	account.ErrMatchUTXO:       {400, "BTM703", "Not found UTXO with given hash"},
	ErrBadActionType:           {400, "BTM704", "Invalid action type"},
	ErrBadAction:               {400, "BTM705", "Invalid action object"},
	ErrBadActionConstruction:   {400, "BTM706", "Invalid action construction"},
	txbuilder.ErrMissingFields: {400, "BTM707", "One or more fields are missing"},
	txbuilder.ErrBadAmount:     {400, "BTM708", "Invalid asset amount"},
	account.ErrFindAccount:     {400, "BTM709", "Not found account"},
	asset.ErrFindAsset:         {400, "BTM710", "Not found asset"},

	// Submit transaction error namespace (73x)
	vm.ErrRunLimitExceeded:      {400, "BTM730", "The BTM Fee is insufficient"},
	vm.ErrDataStackUnderflow:    {400, "BTM731", "Data stack underflow"},
	vm.ErrFalseVMResult:         {400, "BTM732", "Execution of virtual machine failure"},
	validation.ErrNotStandardTx: {400, "BTM733", "Not standard transaction"},
	validation.ErrOverGasCredit: {400, "BTM734", "Gas credit has been spent"},

	// Mock HSM error namespace (8xx)
	pseudohsm.ErrInvalidAfter:         {400, "BTM801", "Invalid `after` in query"},
	pseudohsm.ErrTooManyAliasesToList: {400, "BTM802", "Too many aliases to list"},
	pseudohsm.ErrDuplicateKeyAlias:    {400, "BTM803", "Key Alias already exists"},
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
		context.DeadlineExceeded:     {408, "BTM001", "Request timed out"},
		httpjson.ErrBadRequest:       {400, "BTM003", "Invalid request body"},
		rpc.ErrWrongNetwork:          {502, "BTM104", "A peer core is operating on a different blockchain network"},
		protocol.ErrTheDistantFuture: {400, "BTM105", "Requested height is too far ahead"},

		//accesstoken authz err namespace (86x)
		errNotAuthenticated: {401, "BTM860", "Request could not be authenticated"},
	},
}
