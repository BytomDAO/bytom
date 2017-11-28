package blockchain

import (
	"encoding/json"
)

type JSendResponse struct {
	Status string      `json:"status,omitempty"`
	Msg    string      `json:"msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func JSendWrapper(data interface{}, status, msg string) []byte {
	response := &JSendResponse{
		Status: status,
		Msg:    msg,
		Data:   data,
	}
	rawResponse, err := json.Marshal(response)
	if err != nil {
		return DefaultRawResponse
	}
	return rawResponse
}
