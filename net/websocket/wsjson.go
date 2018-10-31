package websocket

type WSRequest struct {
	Method string      `json:"method"`
	Data   interface{} `json:"Data"`
}

func NewWSRequest(method string, data interface{}) (*WSRequest, error) {

	return &WSRequest{
		Method: method,
		Data:   data,
	}, nil
}
