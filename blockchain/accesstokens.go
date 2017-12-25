package blockchain

import (
	"context"
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
)

var errCurrentToken = errors.New("token cannot delete itself")

func (br *BlockchainReactor) createAccessToken(ctx context.Context, x struct{ ID, Type string }) Response {
	token, err := br.accessTokens.Create(ctx, x.ID, x.Type)
	if err != nil {
		return resWrapper(nil, err)
	}
	data := map[string]*string{"accessToken": token}
	return resWrapper(data)
}

func (br *BlockchainReactor) listAccessTokens(ctx context.Context) Response {
	tokens, err := br.accessTokens.List()
	if err != nil {
		log.Errorf("listAccessTokens: %v", err)
		return resWrapper(nil, err)
	}

	return resWrapper(tokens)
}

func (br *BlockchainReactor) deleteAccessToken(ctx context.Context, x struct{ ID, Token string }) Response {
	//TODO Add delete permission verify.
	if err := br.accessTokens.Delete(ctx, x.ID); err != nil {
		return resWrapper(nil, err)
	}
	return resWrapper(nil)
}

func (br *BlockchainReactor) checkAccessToken(ctx context.Context, x struct{ ID, Secret string }) Response {
	secret, err := hex.DecodeString(x.Secret)
	if err != nil {
		return resWrapper(nil, err)
	}
	_, err = br.accessTokens.Check(ctx, x.ID, secret)
	if err != nil {
		return resWrapper(nil, err)
	}

	return resWrapper(nil)
}
