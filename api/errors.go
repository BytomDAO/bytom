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
	// Build transaction error namespace (70x ~ 72x)
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

	// Submit transaction error namespace (73x ~ 79x)
	// Validation error (73x ~ 75x)
	validation.ErrTxVersion:                 {400, "BTM730", "Invalid transaction version"},
	validation.ErrWrongTransactionSize:      {400, "BTM731", "Invalid transaction size"},
	validation.ErrBadTimeRange:              {400, "BTM732", "Invalid transaction time range"},
	validation.ErrNotStandardTx:             {400, "BTM733", "Not standard transaction"},
	validation.ErrWrongCoinbaseTransaction:  {400, "BTM734", "Invalid coinbase transaction"},
	validation.ErrWrongCoinbaseAsset:        {400, "BTM735", "Invalid coinbase assetID"},
	validation.ErrCoinbaseArbitraryOversize: {400, "BTM736", "Invalid coinbase arbitrary size"},
	validation.ErrEmptyResults:              {400, "BTM737", "No results in the transaction"},
	validation.ErrMismatchedAssetID:         {400, "BTM738", "Mismatched assetID"},
	validation.ErrMismatchedPosition:        {400, "BTM739", "Mismatched value source/dest position"},
	validation.ErrMismatchedReference:       {400, "BTM740", "Mismatched reference"},
	validation.ErrMismatchedValue:           {400, "BTM741", "Mismatched value"},
	validation.ErrMissingField:              {400, "BTM742", "Missing required field"},
	validation.ErrNoSource:                  {400, "BTM743", "No source for value"},
	validation.ErrOverflow:                  {400, "BTM744", "Arithmetic overflow/underflow"},
	validation.ErrPosition:                  {400, "BTM745", "Invalid source or destination position"},
	validation.ErrUnbalanced:                {400, "BTM746", "Unbalanced asset amount between input and output"},
	validation.ErrOverGasCredit:             {400, "BTM747", "Gas credit has been spent"},
	validation.ErrGasCalculate:              {400, "BTM748", "Gas usage calculate got a math error"},

	// VM error (76x ~ 78x)
	vm.ErrAltStackUnderflow:  {400, "BTM760", "Alt stack underflow"},
	vm.ErrBadValue:           {400, "BTM761", "Bad value"},
	vm.ErrContext:            {400, "BTM762", "Wrong context"},
	vm.ErrDataStackUnderflow: {400, "BTM763", "Data stack underflow"},
	vm.ErrDisallowedOpcode:   {400, "BTM764", "Disallowed opcode"},
	vm.ErrDivZero:            {400, "BTM765", "Division by zero"},
	vm.ErrFalseVMResult:      {400, "BTM766", "False result for executing VM"},
	vm.ErrLongProgram:        {400, "BTM767", "Program size exceeds max int32"},
	vm.ErrRange:              {400, "BTM768", "Arithmetic range error"},
	vm.ErrReturn:             {400, "BTM769", "RETURN executed"},
	vm.ErrRunLimitExceeded:   {400, "BTM770", "Run limit exceeded because the BTM Fee is insufficient"},
	vm.ErrShortProgram:       {400, "BTM771", "Unexpected end of program"},
	vm.ErrToken:              {400, "BTM772", "Unrecognized token"},
	vm.ErrUnexpected:         {400, "BTM773", "Unexpected error"},
	vm.ErrUnsupportedVM:      {400, "BTM774", "Unsupported VM because the version of VM is mismatched"},
	vm.ErrVerifyFailed:       {400, "BTM775", "VERIFY failed"},

	// Mock HSM error namespace (8xx)
	pseudohsm.ErrDuplicateKeyAlias:    {400, "BTM800", "Key Alias already exists"},
	pseudohsm.ErrInvalidAfter:         {400, "BTM801", "Invalid `after` in query"},
	pseudohsm.ErrLoadKey:              {400, "BTM802", "Key not found or wrong password"},
	pseudohsm.ErrTooManyAliasesToList: {400, "BTM803", "Requested key aliases exceeds limit"},
	pseudohsm.ErrDecrypt:              {400, "BTM804", "Could not decrypt key with given passphrase"},
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
		httpjson.ErrBadRequest:       {400, "BTM002", "Invalid request body"},
		rpc.ErrWrongNetwork:          {502, "BTM103", "A peer core is operating on a different blockchain network"},
		protocol.ErrTheDistantFuture: {400, "BTM104", "Requested height is too far ahead"},

		//accesstoken authz err namespace (86x)
		errNotAuthenticated: {401, "BTM860", "Request could not be authenticated"},
	},
}
