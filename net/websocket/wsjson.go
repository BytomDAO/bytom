package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/bytom/errors"
)

// IsValidIDType checks that the ID field (which can go in any of the websocket
// requests, responses, or notifications) is valid.
func IsValidIDType(id interface{}) bool {
	switch id.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		string,
		nil:
		return true
	default:
		return false
	}
}

type WSRequest struct {
	Method string      `json:"method"`
	Data   interface{} `json:"Data"`
	ID     interface{} `json:"id"`
}

func NewWSRequest(id interface{}, method string, data interface{}) (*WSRequest, error) {
	if !IsValidIDType(id) {
		str := fmt.Sprintf("the id of type '%T' is invalid", id)
		return nil, errors.New(str)
	}

	return &WSRequest{
		ID:     id,
		Method: method,
		Data:   data,
	}, nil
}

type WSErrResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *WSError        `json:"error"`
	ID     *interface{}    `json:"id"`
}

func NewResponse(id interface{}, marshalledResult []byte, rpcErr *WSError) (*WSErrResponse, error) {
	if !IsValidIDType(id) {
		str := fmt.Sprintf("the id of type '%T' is invalid", id)
		return nil, errors.New(str)
	}

	return &WSErrResponse{
		Result: marshalledResult,
		Error:  rpcErr,
		ID:     &id,
	}, nil
}

func MarshalResponse(id interface{}, result interface{}, rpcErr *WSError) ([]byte, error) {
	marshalledResult, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	response, err := NewResponse(id, marshalledResult, rpcErr)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&response)
}
