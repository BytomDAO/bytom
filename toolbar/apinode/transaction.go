package apinode

import (
	"encoding/hex"
	"encoding/json"

	"github.com/bytom/bytom/blockchain/txbuilder"
	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

type SpendAccountAction struct {
	AccountID string `json:"account_id"`
	*bc.AssetAmount
}

func (s *SpendAccountAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type      string `json:"type"`
		AccountID string `json:"account_id"`
		*bc.AssetAmount
	}{
		Type:        "spend_account",
		AccountID:   s.AccountID,
		AssetAmount: s.AssetAmount,
	})
}

type ControlAddressAction struct {
	Address string `json:"address"`
	*bc.AssetAmount
}

func (c *ControlAddressAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type    string `json:"type"`
		Address string `json:"address"`
		*bc.AssetAmount
	}{
		Type:        "control_address",
		Address:     c.Address,
		AssetAmount: c.AssetAmount,
	})
}

type RetireAction struct {
	*bc.AssetAmount
	Arbitrary []byte
}

func (r *RetireAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Type      string `json:"type"`
		Arbitrary string `json:"arbitrary"`
		*bc.AssetAmount
	}{
		Type:        "retire",
		Arbitrary:   hex.EncodeToString(r.Arbitrary),
		AssetAmount: r.AssetAmount,
	})
}

func (n *Node) BatchSendBTM(accountID, password string, outputs map[string]uint64, memo []byte) error {
	totalBTM := uint64(10000000)
	actions := []interface{}{}
	if len(memo) > 0 {
		actions = append(actions, &RetireAction{
			Arbitrary:   memo,
			AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: 1},
		})
	}

	for address, amount := range outputs {
		actions = append(actions, &ControlAddressAction{
			Address:     address,
			AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: amount},
		})
		totalBTM += amount
	}

	actions = append(actions, &SpendAccountAction{
		AccountID:   accountID,
		AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: totalBTM},
	})

	tpls, err := n.buildTx(actions)
	if err != nil {
		return err
	}

	tpls, err = n.signTx(tpls, password)
	if err != nil {
		return err
	}

	for _, tpl := range tpls {
		if _, err := n.SubmitTx(tpl.Transaction); err != nil {
			return err
		}
	}

	return nil
}

type buildTxReq struct {
	Actions []interface{} `json:"actions"`
}

func (n *Node) buildTx(actions []interface{}) ([]*txbuilder.Template, error) {
	url := "/build-chain-transactions"
	payload, err := json.Marshal(&buildTxReq{Actions: actions})
	if err != nil {
		return nil, errors.Wrap(err, "Marshal spend request")
	}

	result := []*txbuilder.Template{}
	return result, n.request(url, payload, &result)
}

type signTxReq struct {
	Txs      []*txbuilder.Template `json:"transactions"`
	Password string                `json:"password"`
}

type signTxResp struct {
	Txs          []*txbuilder.Template `json:"transaction"`
	SignComplete bool                  `json:"sign_complete"`
}

func (n *Node) signTx(tpls []*txbuilder.Template, password string) ([]*txbuilder.Template, error) {
	url := "/sign-transactions"
	payload, err := json.Marshal(&signTxReq{Txs: tpls, Password: password})
	if err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}

	resp := &signTxResp{}
	if err := n.request(url, payload, resp); err != nil {
		return nil, err
	}

	if !resp.SignComplete {
		return nil, errors.New("sign fail")
	}

	return resp.Txs, nil
}

type submitTxReq struct {
	Tx *types.Tx `json:"raw_transaction"`
}

type submitTxResp struct {
	TxID string `json:"tx_id"`
}

func (n *Node) SubmitTx(tx *types.Tx) (string, error) {
	url := "/submit-transaction"
	payload, err := json.Marshal(submitTxReq{Tx: tx})
	if err != nil {
		return "", errors.Wrap(err, "json marshal")
	}

	res := &submitTxResp{}
	return res.TxID, n.request(url, payload, res)
}
