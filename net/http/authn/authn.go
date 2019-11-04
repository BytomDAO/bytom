package authn

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytom/bytom/accesstoken"
	"github.com/bytom/bytom/errors"
)

const tokenExpiry = time.Minute * 5

var loopbackOn = true

var (
	//ErrInvalidToken is returned when authenticate is called with invalid token.
	ErrInvalidToken = errors.New("invalid token")
	//ErrNoToken is returned when authenticate is called with no token.
	ErrNoToken = errors.New("no token")
)

//API describe the token authenticate.
type API struct {
	disable  bool
	tokens   *accesstoken.CredentialStore
	tokenMu  sync.Mutex // protects the following
	tokenMap map[string]tokenResult
}

type tokenResult struct {
	lastLookup time.Time
}

//NewAPI create a token authenticate object.
func NewAPI(tokens *accesstoken.CredentialStore, disable bool) *API {
	return &API{
		disable:  disable,
		tokens:   tokens,
		tokenMap: make(map[string]tokenResult),
	}
}

// Authenticate returns the request, with added tokens and/or localhost
// flags in the context, as appropriate.
func (a *API) Authenticate(req *http.Request) (*http.Request, error) {
	ctx := req.Context()

	token, err := a.tokenAuthn(req)
	if err == nil && token != "" {
		// if this request was successfully authenticated with a token, pass the token along
		ctx = newContextWithToken(ctx, token)
	}

	local := a.localhostAuthn(req)
	if local {
		ctx = newContextWithLocalhost(ctx)
	}

	if !local && strings.HasPrefix(req.URL.Path, "/backup-wallet") {
		return req.WithContext(ctx), errors.New("only local can get access backup-wallets")
	}

	if !local && strings.HasPrefix(req.URL.Path, "/restore-wallet") {
		return req.WithContext(ctx), errors.New("only local can get access restore-wallet")
	}

	if !local && strings.HasPrefix(req.URL.Path, "/list-access-tokens") {
		return req.WithContext(ctx), errors.New("only local can get access token list")
	}

	// Temporary workaround. Dashboard is always ok.
	// See loopbackOn comment above.
	if strings.HasPrefix(req.URL.Path, "/dashboard/") || req.URL.Path == "/dashboard" {
		return req.WithContext(ctx), nil
	}
	// Adding this workaround for Equity Playground.
	if strings.HasPrefix(req.URL.Path, "/equity/") || req.URL.Path == "/equity" {
		return req.WithContext(ctx), nil
	}
	if loopbackOn && local {
		return req.WithContext(ctx), nil
	}

	return req.WithContext(ctx), err
}

// returns true if this request is coming from a loopback address
func (a *API) localhostAuthn(req *http.Request) bool {
	h, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return false
	}
	if !net.ParseIP(h).IsLoopback() {
		return false
	}
	return true
}

func (a *API) tokenAuthn(req *http.Request) (string, error) {
	if a.disable {
		return "", nil
	}

	user, pw, ok := req.BasicAuth()
	if !ok {
		return "", ErrNoToken
	}
	return user, a.cachedTokenAuthnCheck(req.Context(), user, pw)
}

func (a *API) cachedTokenAuthnCheck(ctx context.Context, user, pw string) error {
	a.tokenMu.Lock()
	res, ok := a.tokenMap[user+pw]
	a.tokenMu.Unlock()
	if !ok || time.Now().After(res.lastLookup.Add(tokenExpiry)) {
		err := a.tokens.Check(user, pw)
		if err != nil {
			return ErrInvalidToken
		}
		res = tokenResult{lastLookup: time.Now()}
		a.tokenMu.Lock()
		a.tokenMap[user+pw] = res
		a.tokenMu.Unlock()
	}
	return nil
}
