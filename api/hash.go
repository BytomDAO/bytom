package api

import (
	"crypto/sha256"

	"golang.org/x/crypto/sha3"

	chainjson "github.com/bytom/encoding/json"
)

// operation of sha3
func (a *API) calculateSha3(ins struct {
	Data string `json:"data"`
}) Response {
	hash := sha3.Sum256([]byte(ins.Data))
	return NewSuccessResponse(map[string]chainjson.HexBytes{"sha3_result": hash[:]})
}

// operation of sha256
func (a *API) calculateSha256(ins struct {
	Data string `json:"data"`
}) Response {
	hash := sha256.Sum256([]byte(ins.Data))
	return NewSuccessResponse(map[string]chainjson.HexBytes{"sha256_result": hash[:]})
}
