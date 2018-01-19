package txfeed

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm/vmutil"
)

const (
	//FilterNumMax max txfeed filter amount.
	FilterNumMax = 1024
)

var (
	//ErrDuplicateAlias means error of duplicate feed alias.
	ErrDuplicateAlias = errors.New("duplicate feed alias")
	//ErrEmptyAlias means error of empty feed alias.
	ErrEmptyAlias = errors.New("empty feed alias")
	//ErrNumExceedlimit means txfeed filter number exceeds the limit.
	ErrNumExceedlimit  = errors.New("txfeed exceed limit")
	maxNewTxfeedChSize = 1000
)

//Tracker filter tracker object.
type Tracker struct {
	DB       dbm.DB
	TxFeeds  []*TxFeed
	chain    *protocol.Chain
	txfeedCh chan *legacy.Tx
}

type rawOutput struct {
	OutputID bc.Hash
	bc.AssetAmount
	ControlProgram []byte
	txHash         bc.Hash
	outputIndex    uint32
	sourceID       bc.Hash
	sourcePos      uint64
	refData        bc.Hash
}

//TxFeed describe a filter
type TxFeed struct {
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias"`
	Filter string `json:"filter,omitempty"`
	Param  filter `json:"param,omitempty"`
}

type filter struct {
	assetID          string `json:"assetid,omitempty"`
	amountLowerLimit uint64 `json:"lowerlimit,omitempty"`
	amountUpperLimit uint64 `json:"upperlimit,omitempty"`
	transType        string `json:"transtype,omitempty"`
}

//NewTracker create new txfeed tracker.
func NewTracker(db dbm.DB, chain *protocol.Chain) *Tracker {
	s := &Tracker{
		DB:       db,
		TxFeeds:  make([]*TxFeed, 0, 10),
		chain:    chain,
		txfeedCh: make(chan *legacy.Tx, maxNewTxfeedChSize),
	}

	return s
}

func loadTxFeed(db dbm.DB, txFeeds []*TxFeed) ([]*TxFeed, error) {
	iter := db.Iterator()
	defer iter.Release()

	for iter.Next() {
		txFeed := &TxFeed{}
		if err := json.Unmarshal(iter.Value(), &txFeed); err != nil {
			return nil, err
		}
		filter, err := parseFilter(txFeed.Filter)
		if err != nil {
			return nil, err
		}
		txFeed.Param = filter
		txFeeds = append(txFeeds, txFeed)
	}
	return txFeeds, nil
}

func parseFilter(ft string) (filter, error) {
	var res filter

	subFilter := strings.Split(ft, "AND")
	for _, value := range subFilter {
		param := getParam(value, "=")
		if param == "" {
			continue
		}
		if strings.Contains(value, "asset_id") {
			res.assetID = param
		}
		if strings.Contains(value, "amount_lower_limit") {
			tmp, _ := strconv.ParseInt(param, 10, 64)
			res.amountLowerLimit = uint64(tmp)
		}
		if strings.Contains(value, "amount_upper_limit") {
			tmp, _ := strconv.ParseInt(param, 10, 64)
			res.amountUpperLimit = uint64(tmp)
		}
		if strings.Contains(value, "trans_type") {
			res.transType = param
		}
	}
	return res, nil
}

//TODO
func getParam(str, substr string) string {
	if result := strings.Index(str, substr); result >= 0 {
		str := strings.Replace(str[result+1:], "'", "", -1)
		str = strings.Replace(str, " ", "", -1)
		return str
	}
	return ""
}

func parseTxfeed(db dbm.DB, filters []filter) error {
	var txFeed TxFeed
	var index int

	iter := db.Iterator()
	defer iter.Release()

	for iter.Next() {

		if err := json.Unmarshal(iter.Value(), &txFeed); err != nil {
			return err
		}

		subFilter := strings.Split(txFeed.Filter, "AND")
		for _, value := range subFilter {
			param := getParam(value, "=")
			if param == "" {
				continue
			}
			if strings.Contains(value, "asset_id") {
				filters[index].assetID = param
			}
			if strings.Contains(value, "amount_lower_limit") {
				tmp, _ := strconv.ParseInt(param, 10, 64)
				filters[index].amountLowerLimit = uint64(tmp)
			}
			if strings.Contains(value, "amount_upper_limit") {
				tmp, _ := strconv.ParseInt(param, 10, 64)
				filters[index].amountUpperLimit = uint64(tmp)
			}
			if strings.Contains(value, "trans_type") {
				filters[index].transType = param
			}
		}
		index++
	}
	return nil
}

//Prepare load and parse filters.
func (t *Tracker) Prepare(ctx context.Context) error {
	var err error
	t.TxFeeds, err = loadTxFeed(t.DB, t.TxFeeds)
	return err
}

//GetTxfeedCh return a txfeed channel.
func (t *Tracker) GetTxfeedCh() chan *legacy.Tx {
	return t.txfeedCh
}

//Create create a txfeed filter.
func (t *Tracker) Create(ctx context.Context, alias, fil string) error {
	// Validate the filter.

	if err := query.ValidateTransactionFilter(fil); err != nil {
		return err
	}

	if alias == "" {
		return errors.WithDetail(ErrEmptyAlias, "a transaction feed with empty alias")
	}

	if len(t.TxFeeds) >= FilterNumMax {
		return errors.WithDetail(ErrNumExceedlimit, "txfeed number exceed limit")
	}

	for _, txfeed := range t.TxFeeds {
		if txfeed.Alias == alias {
			return errors.WithDetail(ErrDuplicateAlias, "txfeed alias must unique")
		}
	}

	feed := &TxFeed{
		Alias:  alias,
		Filter: fil,
	}

	filter, err := parseFilter(feed.Filter)
	if err != nil {
		return err
	}
	feed.Param = filter
	t.TxFeeds = append(t.TxFeeds, feed)
	return insertTxFeed(t.DB, feed)
}

func deleteTxFeed(db dbm.DB, alias string) error {
	key, err := json.Marshal(alias)
	if err != nil {
		return err
	}
	db.Delete(key)
	return nil
}

// insertTxFeed adds the txfeed to the database. If the txfeed has a client token,
// and there already exists a txfeed with that client token, insertTxFeed will
// lookup and return the existing txfeed instead.
func insertTxFeed(db dbm.DB, feed *TxFeed) error {
	// var err error
	key, err := json.Marshal(feed.Alias)
	if err != nil {
		return err
	}
	value, err := json.Marshal(feed)
	if err != nil {
		return err
	}

	db.Set(key, value)
	return nil
}

//Get get txfeed filter with alias.
func (t *Tracker) Get(ctx context.Context, alias string) (*TxFeed, error) {
	if alias == "" {
		return nil, errors.WithDetail(ErrEmptyAlias, "get transaction feed with empty alias")
	}

	for i, v := range t.TxFeeds {
		if v.Alias == alias {
			return t.TxFeeds[i], nil
		}
	}
	return nil, nil
}

//Delete delete txfeed with alias.
func (t *Tracker) Delete(ctx context.Context, alias string) error {
	log.WithField("delete", alias).Info("delete txfeed")

	if alias == "" {
		return errors.WithDetail(ErrEmptyAlias, "del transaction feed with empty alias")
	}

	for i, txfeed := range t.TxFeeds {
		if txfeed.Alias == alias {
			t.TxFeeds = append(t.TxFeeds[:i], t.TxFeeds[i+1:]...)
			return deleteTxFeed(t.DB, alias)
		}
	}
	return nil
}

func outputFilter(txfeed *TxFeed, value *query.AnnotatedOutput) bool {
	assetidstr := value.AssetID.String()

	if 0 != strings.Compare(txfeed.Param.assetID, assetidstr) && txfeed.Param.assetID != "" {
		return false
	}
	if 0 != strings.Compare(txfeed.Param.transType, value.Type) && txfeed.Param.transType != "" {
		return false
	}
	if txfeed.Param.amountLowerLimit > value.Amount && txfeed.Param.amountLowerLimit != 0 {
		return false
	}
	if txfeed.Param.amountUpperLimit < value.Amount && txfeed.Param.amountUpperLimit != 0 {
		return false
	}

	return true
}

//TxFilter filter tx from mempool.
func (t *Tracker) TxFilter(tx *legacy.Tx) error {
	var annotatedTx *query.AnnotatedTx
	// Build the fully annotated transaction.
	annotatedTx = buildAnnotatedTransaction(tx)
	for _, output := range annotatedTx.Outputs {
		for _, filter := range t.TxFeeds {
			if match := outputFilter(filter, output); !match {
				continue
			}
			b, err := json.Marshal(annotatedTx)
			if err != nil {
				return err
			}
			log.WithField("filter", string(b)).Info("find new tx match filter")
			t.txfeedCh <- tx
		}
	}
	return nil
}

var emptyJSONObject = json.RawMessage(`{}`)

func buildAnnotatedTransaction(orig *legacy.Tx) *query.AnnotatedTx {
	tx := &query.AnnotatedTx{
		ID:      orig.ID,
		Inputs:  make([]*query.AnnotatedInput, 0, len(orig.Inputs)),
		Outputs: make([]*query.AnnotatedOutput, 0, len(orig.Outputs)),
	}

	for i := range orig.Inputs {
		tx.Inputs = append(tx.Inputs, buildAnnotatedInput(orig, uint32(i)))
	}
	for i := range orig.Outputs {
		tx.Outputs = append(tx.Outputs, buildAnnotatedOutput(orig, i))
	}
	return tx
}

func buildAnnotatedInput(tx *legacy.Tx, i uint32) *query.AnnotatedInput {
	orig := tx.Inputs[i]
	in := &query.AnnotatedInput{
		AssetID:         orig.AssetID(),
		Amount:          orig.Amount(),
		AssetDefinition: &emptyJSONObject,
		ReferenceData:   &emptyJSONObject,
	}

	id := tx.Tx.InputIDs[i]
	e := tx.Entries[id]
	switch e := e.(type) {
	case *bc.Spend:
		in.Type = "spend"
		in.ControlProgram = orig.ControlProgram()
		in.SpentOutputID = e.SpentOutputId
	case *bc.Issuance:
		in.Type = "issue"
		in.IssuanceProgram = orig.IssuanceProgram()
	}

	return in
}

func buildAnnotatedOutput(tx *legacy.Tx, idx int) *query.AnnotatedOutput {
	orig := tx.Outputs[idx]
	outid := tx.OutputID(idx)
	out := &query.AnnotatedOutput{
		OutputID:        *outid,
		Position:        idx,
		AssetID:         *orig.AssetId,
		AssetDefinition: &emptyJSONObject,
		Amount:          orig.Amount,
		ControlProgram:  orig.ControlProgram,
		ReferenceData:   &emptyJSONObject,
	}

	if vmutil.IsUnspendable(out.ControlProgram) {
		out.Type = "retire"
	} else {
		out.Type = "control"
	}
	return out
}
