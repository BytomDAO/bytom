package websocket

import "fmt"

var (
	ErrWSInvalidRequest = &WSError{
		Code:    "BTM32600",
		Message: "Invalid request",
	}
	ErrWSMethodNotFound = &WSError{
		Code:    "BTM32601",
		Message: "Method not found",
	}
	ErrWSInvalidParams = &WSError{
		Code:    "BTM32602",
		Message: "Invalid parameters",
	}
	ErrWSInternal = &WSError{
		Code:    "BTM32603",
		Message: "Internal error",
	}
	ErrWSParse = &WSError{
		Code:    "BTM32700",
		Message: "Parse error",
	}
)

type WSErrorCode string

// WSError represents an error that is used as a part of a websocket Response
// object.
type WSError struct {
	WSStatus int         `json:"-"`
	Code     WSErrorCode `json:"code,omitempty"`
	Message  string      `json:"message,omitempty"`
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
