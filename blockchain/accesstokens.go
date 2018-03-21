package blockchain

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func (bcr *BlockchainReactor) createAccessToken(ctx context.Context, x struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}) Response {
	token, err := bcr.wallet.Tokens.Create(ctx, x.ID, x.Type)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(token)
}

func (bcr *BlockchainReactor) listAccessTokens(ctx context.Context) Response {
	tokens, err := bcr.wallet.Tokens.List(ctx)
	if err != nil {
		log.Errorf("listAccessTokens: %v", err)
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tokens)
}

func (bcr *BlockchainReactor) deleteAccessToken(ctx context.Context, x struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}) Response {
	//TODO Add delete permission verify.
	if err := bcr.wallet.Tokens.Delete(ctx, x.ID); err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(nil)
}

func (bcr *BlockchainReactor) checkAccessToken(ctx context.Context, x struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}) Response {
	if _, err := bcr.wallet.Tokens.Check(ctx, x.ID, x.Secret); err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(nil)
}
