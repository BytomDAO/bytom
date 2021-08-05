package apinode

import (
	"encoding/json"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/toolbar/common"
)

// Node can invoke the api which provide by the full node server
type Node struct {
	hostPort string
}

// NewNode create a api client with target server
func NewNode(hostPort string) *Node {
	return &Node{hostPort: hostPort}
}

type response struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrDetail string          `json:"error_detail"`
}

func (n *Node) request(path string, payload []byte, respData interface{}) error {
	resp := &response{}
	if err := common.Post(n.hostPort+path, payload, resp); err != nil {
		return err
	}

	if resp.Status != "success" {
		return errors.New(resp.ErrDetail)
	}

	if resp.Data == nil {
		return nil
	}

	return json.Unmarshal(resp.Data, respData)
}
