package websocket

// WSRequest means the data structure of the request
type WSRequest struct {
	Topic string `json:"topic"`
}

// NewWSRequest creates a request data object
func NewWSRequest(topic string) *WSRequest {
	return &WSRequest{
		Topic: topic,
	}
}

// WSResponse means the returned data structure
type WSResponse struct {
	NotificationType string      `json:"notification_type"`
	Data             interface{} `json:"data"`
	ErrorDetail      string      `json:"error_detail,omitempty"`
}

// NewWSResponse creates a return data object
func NewWSResponse(notificationType string, data interface{}, err error) *WSResponse {
	wsResp := &WSResponse{
		NotificationType: notificationType,
		Data:             data,
	}

	if err != nil {
		wsResp.ErrorDetail = err.Error()
	}

	return wsResp
}
