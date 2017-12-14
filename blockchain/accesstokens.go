package blockchain

import (
	"context"
	"encoding/hex"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/errors"
	"github.com/bytom/net/http/httpjson"
)

var errCurrentToken = errors.New("token cannot delete itself")

func (br *BlockchainReactor) createAccessToken(ctx context.Context, x struct{ ID, Type string }) []byte {
	token, err := br.accessTokens.Create(ctx, x.ID, x.Type)
	if err != nil {
		return resWrapper(nil, err)
	}

	data := []string{*token}
	return resWrapper(data)
}

func (br *BlockchainReactor) listAccessTokens(ctx context.Context, query requestQuery) []byte {
	limit := query.PageSize
	if limit == 0 {
		limit = defGenericPageSize
	}
	tokens, after, last, err := br.accessTokens.List(query.After, limit, defGenericPageSize)
	if err != nil {
		log.Errorf("listAccessTokens: %v", err)
		return resWrapper(nil, err)
	}

	query.After = after
	page := &page{
		Items:    httpjson.Array(tokens),
		LastPage: last,
		Next:     query}

	rawPage, err := json.Marshal(page)
	if err != nil {
		return resWrapper(nil, err)
	}

	data := []string{string(rawPage)}
	return resWrapper(data)
}

func (br *BlockchainReactor) deleteAccessToken(ctx context.Context, x struct{ ID, Token string }) []byte {
	//TODO Add delete permission verify.
	if err := br.accessTokens.Delete(ctx, x.ID); err != nil {
		return resWrapper(nil, err)
	}
	return resWrapper(nil)
}

func (br *BlockchainReactor) checkAccessToken(ctx context.Context, x struct{ ID, Secret string }) []byte {
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
