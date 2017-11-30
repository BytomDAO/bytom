package blockchain

import (
	"encoding/json"
)

type jsendResponse struct {
	Status string      `json:"status,omitempty"`
	Msg    string      `json:"msg,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

func jsendWrapper(data interface{}, status, msg string) []byte {
	response := &jsendResponse{
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
