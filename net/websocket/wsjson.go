package websocket

type WSRequest struct {
	Topic string `json:"topic"`
}

func NewWSRequest(topic string) *WSRequest {
	return &WSRequest{
		Topic: topic,
	}
}

type WSResponse struct {
	NotificationType string      `json:"notification_type"`
	Data             interface{} `json:"data"`
	Error            string      `json:"error"`
}

func NewWSResponse(notificationType string, data interface{}, err string) *WSResponse {
	return &WSResponse{
		NotificationType: notificationType,
		Data:             data,
		Error:            err,
	}
}
