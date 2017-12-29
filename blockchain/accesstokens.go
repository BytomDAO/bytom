package blockchain

import (
	"context"
	"encoding/hex"

	"github.com/bytom/errors"
)

var errCurrentToken = errors.New("token cannot delete itself")

func (br *BlockchainReactor) createAccessToken(ctx context.Context, x struct{ ID, Type string }) interface{} {
	token, err := br.accessTokens.Create(ctx, x.ID, x.Type)
	if err != nil {
		return jsendWrapper(nil, ERROR, err.Error())
	}

	return jsendWrapper(token, SUCCESS, "")
}

func (br *BlockchainReactor) listAccessTokens(ctx context.Context) interface{} {
	tokens, err := br.accessTokens.List(ctx)
	if err != nil {
		return jsendWrapper(nil, ERROR, err.Error())
	}

	return jsendWrapper(tokens, SUCCESS, "")
}

func (br *BlockchainReactor) deleteAccessToken(ctx context.Context, x struct{ ID, Token string }) interface{} {
	//TODO Add delete permission verify.
	if err := br.accessTokens.Delete(ctx, x.ID); err != nil {
		return jsendWrapper(nil, ERROR, err.Error())
	}
	return jsendWrapper("success", SUCCESS, "")
}

func (br *BlockchainReactor) checkAccessToken(ctx context.Context, x struct{ ID, Secret string }) interface{} {
	secret, err := hex.DecodeString(x.Secret)
	if err != nil {
		return jsendWrapper(nil, ERROR, err.Error())
	}
	result, err := br.accessTokens.Check(ctx, x.ID, secret)
	if err != nil {
		return jsendWrapper(nil, ERROR, err.Error())
	}
	if result == true {
		return jsendWrapper("success", SUCCESS, "")
	}
	return jsendWrapper("fail", SUCCESS, "")
}
