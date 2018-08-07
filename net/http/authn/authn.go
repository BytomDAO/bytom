package authn

import (
	"context"
	"crypto/x509"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytom/accesstoken"
	"github.com/bytom/errors"
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
	tokens             *accesstoken.CredentialStore
	crosscoreRPCPrefix string
	rootCAs            *x509.CertPool

	tokenMu  sync.Mutex // protects the following
	tokenMap map[string]tokenResult
}

type tokenResult struct {
	lastLookup time.Time
}

//NewAPI create a token authenticate object.
func NewAPI(tokens *accesstoken.CredentialStore) *API {
	return &API{
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

// checks the request for a valid client cert list.
// If found, it is added to the request's context.
// Note that an *invalid* client cert is treated the
// same as no client cert -- it is omitted from the
// returned context, but the connection may proceed.
func certAuthn(req *http.Request, rootCAs *x509.CertPool) context.Context {
	if req.TLS != nil && len(req.TLS.PeerCertificates) > 0 {
		certs := req.TLS.PeerCertificates

		// Same logic as serverHandshakeState.processCertsFromClient
		// in $GOROOT/src/crypto/tls/handshake_server.go.
		opts := x509.VerifyOptions{
			Roots:         rootCAs,
			CurrentTime:   time.Now(),
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		for _, cert := range certs[1:] {
			opts.Intermediates.AddCert(cert)
		}

		if _, err := certs[0].Verify(opts); err != nil {
			// crypto/tls treats this as an error:
			// errors.New("tls: failed to verify client's certificate: " + err.Error())
			// For us, it is ok; we want to treat it the same as if there
			// were no client cert presented.
			return req.Context()
		}

		return context.WithValue(req.Context(), x509CertsKey, certs)
	}
	return req.Context()
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
