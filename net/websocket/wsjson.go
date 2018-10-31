package websocket

type WSRequest struct {
	Method string      `json:"method"`
	Data   interface{} `json:"data"`
}

func NewWSRequest(method string, data interface{}) *WSRequest {

	return &WSRequest{
		Method: method,
		Data:   data,
	}
}

type WSResponse struct {
	Method string      `json:"method"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

func NewWSResponse(method string, data interface{}, err string) *WSResponse {

	return &WSResponse{
		Method: method,
		Data:   data,
		Error:  err,
	}
}
