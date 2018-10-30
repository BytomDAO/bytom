package api

import (
	"encoding/json"
	"fmt"

	"github.com/bytom/errors"
)

var (
	ErrWSInvalidRequest = &WSError{
		Code:    -32600,
		Message: "Invalid request",
	}
	ErrWSMethodNotFound = &WSError{
		Code:    -32601,
		Message: "Method not found",
	}
	ErrWSInvalidParams = &WSError{
		Code:    -32602,
		Message: "Invalid parameters",
	}
	ErrWSInternal = &WSError{
		Code:    -32603,
		Message: "Internal error",
	}
	ErrWSParse = &WSError{
		Code:    -32700,
		Message: "Parse error",
	}
)

type WSErrorCode int

// WSError represents an error that is used as a part of a websocket Response
// object.
type WSError struct {
	Code    WSErrorCode `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
}

var _, _ error = WSError{}, (*WSError)(nil)

func (e WSError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// NewWSError constructs and returns a new JSON-RPC error that is suitable
// for use in a websocket Response object.
func NewWSError(code WSErrorCode, message string) *WSError {
	return &WSError{
		Code:    code,
		Message: message,
	}
}

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
