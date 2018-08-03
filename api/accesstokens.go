package api

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func (a *API) createAccessToken(ctx context.Context, x struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}) Response {
	token, err := a.accessTokens.Create(x.ID, x.Type)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(token)
}

func (a *API) listAccessTokens(ctx context.Context) Response {
	tokens, err := a.accessTokens.List()
	if err != nil {
		log.Errorf("listAccessTokens: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tokens)
}

func (a *API) deleteAccessToken(ctx context.Context, x struct {
	ID string `json:"id"`
}) Response {
	//TODO Add delete permission verify.
	if err := a.accessTokens.Delete(x.ID); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

func (a *API) checkAccessToken(ctx context.Context, x struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}) Response {
	if err := a.accessTokens.Check(x.ID, x.Secret); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}
