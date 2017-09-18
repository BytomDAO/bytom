//+build !no_Pseudohsm

package core

import (
	"context"
	"bytom/core/pseudohsm"
	"bytom/core/txbuilder"
	"bytom/crypto/ed25519/chainkd"
	"bytom/net/http/httperror"
	"bytom/net/http/httpjson"
)

func init() {
	errorFormatter.Errors[Pseudohsm.ErrDuplicateKeyAlias] = httperror.Info{400, "BTM050", "Alias already exists"}
	errorFormatter.Errors[Pseudohsm.ErrInvalidAfter] = httperror.Info{400, "BTM801", "Invalid `after` in query"}
	errorFormatter.Errors[Pseudohsm.ErrTooManyAliasesToList] = httperror.Info{400, "BTM802", "Too many aliases to list"}
}

type requestQuery struct {
	Filter       string        `json:"filter,omitempty"`
	FilterParams []interface{} `json:"filter_params,omitempty"`
	SumBy        []string      `json:"sum_by,omitempty"`
	PageSize     int           `json:"page_size"`

	// AscLongPoll and Timeout are used by /list-transactions
	// to facilitate notifications.
	AscLongPoll bool          `json:"ascending_with_long_poll,omitempty"`
	Timeout     json.Duration `json:"timeout"`

	// After is a completely opaque cursor, indicating that only
	// items in the result set after the one identified by `After`
	// should be included. It has no relationship to time.
	After string `json:"after"`

	// These two are used for time-range queries like /list-transactions
	StartTimeMS uint64 `json:"start_time,omitempty"`
	EndTimeMS   uint64 `json:"end_time,omitempty"`

	// This is used for point-in-time queries like /list-balances
	// TODO(bobg): Different request structs for endpoints with different needs
	TimestampMS uint64 `json:"timestamp,omitempty"`

	// This is used for filtering results from /list-access-tokens
	// Value must be "client" or "network"
	Type string `json:"type"`

	// Aliases is used to filter results from /mockshm/list-keys
	Aliases []string `json:"aliases,omitempty"`
}


// PseudoHSM configures the Core to expose the PseudoHSM endpoints. It
// is only included in non-production builds.
func PseudoHSM(hsm *Pseudohsm.HSM) RunOption {
	return func(api *API) {

		h := &pseudoHSMHandler{PseudoHSM: hsm}
		needConfig := api.needConfig()
		api.mux.Handle("/hsm/create-key", needConfig(h.pseudohsmCreateKey))
		api.mux.Handle("/hsm/list-keys", needConfig(h.pseudohsmListKeys))
		api.mux.Handle("/hsm/delete-key", needConfig(h.pseudohsmDeleteKey))
		api.mux.Handle("/hsm/sign-transaction", needConfig(h.pseudohsmSignTemplates))
		api.mux.Handle("/hsm/reset-password", needConfig(h.pseudohsmResetPassword))
		api.mux.Handle("/hsm/update-alias", needConfig(h.pseudohsmUpdateAlias))


	}
}



type pseudoHSMHandler struct {
	PseudoHSM *Pseudohsm.HSM
}

func (h *PseudoHSMHandler) pseudohsmCreateKey(password string, in struct{ Alias string }) (result *Pseudohsm.XPub, err error) {
	return h.PseudoHSM.XCreate(password, in.Alias)
}

func (h *PseudoHSMHandler) pseudohsmListKeys(query requestQuery) (page, error) {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize  // defGenericPageSize = 100
	}

	xpubs, after, err := h.PseudoHSM.ListKeys(query.After, limit)
	if err != nil {
		return page{}, err
	}

	var items []interface{}
	for _, xpub := range xpubs {
		items = append(items, xpub)
	}

	query.After = after

	return page{
		Items:    httpjson.Array(items),
		LastPage: len(xpubs) < limit,
		Next:     query,
	}, nil
}

func (h *PseudoHSMHandler) pseudohsmDeleteKey(xpub chainkd.XPub, password string) error {
	return h.PseudoHSM.XDelete(xpub, password)
}

func (h *PseudoHSMHandler) pseudohsmSignTemplates(x struct {
	Txs   []*txbuilder.Template `json:"transactions"`
	XPubs []chainkd.XPub        `json:"xpubs"`
}) []interface{} {
	resp := make([]interface{}, 0, len(x.Txs))
	for _, tx := range x.Txs {
		err := txbuilder.Sign(tx, x.XPubs, h.pseudohsmSignTemplate)
		if err != nil {
			info := errorFormatter.Format(err)
			resp = append(resp, info)
		} else {
			resp = append(resp, tx)
		}
	}
	return resp
}

func (h *PseudoHSMHandler) pseudohsmSignTemplate(xpub chainkd.XPub, path [][]byte, data [32]byte) ([]byte, error) {
	sigBytes, err := h.PseudoHSM.XSign(xpub, path, data[:])
	if err == Pseudohsm.ErrNoKey {
		return nil, nil
	}
	return sigBytes, err
}

// remote hsm used

func RemoteHSM(hsm *remoteHSM) RunOption {
	return func(api *API) {
		h := &retmoteHSMHandler{RemoteHSM: hsm}
		needConfig := api.needConfig()
		api.mux.Handle("/hsm/sign-transaction", needConfig(h.Sign))
	}
}

/*
type remoteHSM struct {
	Client *rpc.Client
}

func remoteHSMHandler struct {
	RemoteHSM  *remoteHSM
}

func New(conf *config.Config) *HSM {

	httpClient := new(http.Client)
	httpClient.Transport = &http.Transport{
		TLSClientConfig: tlsConfig,
		// The following fields are default values
		// copied from DefaultTransport.
		// (When you change them, be sure to move them
		// above this line so this comment stays true.)
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &remoteHSM{Client: &rpc.Client{
			BaseURL:      conf.HsmUrl,
			AccessToken:  conf.HsmAccessToken,
			Username:     conf.processID,
			CoreID:       conf.Id,
			Version:      conf.version,
			BlockchainID: conf.BlockchainId.String(),
			Client:       httpClient,
		}}
}


func (h *remoteHSM) Sign(ctx context.Context, pk ed25519.PublicKey, date [32]byte)([]byte, err error) {
	body := struct {
		Block *legacy.TxHeader 	  `json:"txheader"`
		Pub   json.HexBytes       `json:"pubkey"`
	}{data, json.HexBytes(pk[:])}

	err = h.Client.Call(ctx, "/sign-transaction", body, &sigBytes)
	return sigBytes
}
*/