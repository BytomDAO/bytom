package apinode

import (
	"encoding/json"

	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/netsync/peers"
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

func (n *Node) DisconnectPeer(peerID string) error {
	url := "/disconnect-peer"
	payload, err := json.Marshal(struct {
		PeerID string `json:"peer_id"`
	}{
		PeerID: peerID,
	})
	if err != nil {
		return err
	}

	return n.request(url, payload, nil)

}

func (n *Node) ConnectPeer(ip string, port uint16) (*peers.Peer, error) {
	url := "/connect-peer"
	payload, err := json.Marshal(struct {
		Ip   string `json:"ip"`
		Port uint16 `json:"port"`
	}{
		Ip:   ip,
		Port: port,
	})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	res := &peers.Peer{}
	return res, n.request(url, payload, res)
}
