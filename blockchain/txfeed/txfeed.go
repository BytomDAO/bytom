package txfeed

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/bytom/blockchain/query"
	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
	"github.com/bytom/protocol/vm/vmutil"
	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"
)

const (
	FilterNumMax = 1024
)

var (
	ErrDuplicateAlias  = errors.New("duplicate feed alias")
	ErrEmptyAlias      = errors.New("empty feed alias")
	ErrNumExceedlimit  = errors.New("txfeed exceed limit")
	maxNewTxfeedChSize = 1000
)

type Tracker struct {
	DB                dbm.DB
	TxFeeds           []TxFeed
	BlockTransactions map[bc.Hash]*legacy.Tx
	chain             *protocol.Chain
	txfeedCh          chan *legacy.Tx
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

type TxFeed struct {
	ID     string `json:"id,omitempty"`
	Alias  string `json:"alias"`
	Filter string `json:"filter,omitempty"`
	After  string `json:"after,omitempty"`
	Param  filter `json:"param,omitempty"`
}

type filter struct {
	assetID          string `json:"assetid,omitempty"`
	amountLowerLimit uint64 `json:"lowerlimit,omitempty"`
	amountUpperLimit uint64 `json:"upperlimit,omitempty"`
	transType        string `json:"transtype,omitempty"`
}

//NewTracker create new txfeed tracker
func NewTracker(db dbm.DB, chain *protocol.Chain) *Tracker {
	s := &Tracker{
		DB:                db,
		TxFeeds:           make([]TxFeed, 0, 10),
		BlockTransactions: make(map[bc.Hash]*legacy.Tx),
		chain:             chain,
		txfeedCh:          make(chan *legacy.Tx, maxNewTxfeedChSize),
	}

	return s
}

func loadTxFeed(db dbm.DB, txFeeds []TxFeed) ([]TxFeed, error) {
	var txFeed = TxFeed{}
	iter := db.Iterator()
	for iter.Next() {
		err := json.Unmarshal(iter.Value(), &txFeed)
		if err != nil {
			return nil, err
		}
		filter, err := parseFilter(txFeed.Filter)
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

func getParam(str, substr string) string {
	result := strings.Index(str, substr)
	if result >= 0 {
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
	for iter.Next() {
		err := json.Unmarshal(iter.Value(), &txFeed)
		if err != nil {
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

//Prepare load and parse filters
func (t *Tracker) Prepare(ctx context.Context) error {
	var err error
	t.TxFeeds, err = loadTxFeed(t.DB, t.TxFeeds)
	log.WithField("prepare", t.TxFeeds).Info("load txfeed")
	if err != nil {
		return err
	}

	return nil
}

// GetTxfeedCh return a txfeed channel
func (t *Tracker) GetTxfeedCh() chan *legacy.Tx {
	return t.txfeedCh
}

//Create create a txfeed filter
func (t *Tracker) Create(ctx context.Context, alias, fil, after string) error {
	// Validate the filter.
	err := query.ValidateTransactionFilter(fil)
	if err != nil {
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

	feed := TxFeed{
		Alias:  alias,
		Filter: fil,
		After:  after,
	}

	filter, err := parseFilter(feed.Filter)
	if err != nil {
		return err
	}
	feed.Param = filter
	t.TxFeeds = append(t.TxFeeds, feed)
	return insertTxFeed(t.DB, &feed)
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

func (t *Tracker) Find(ctx context.Context, id, alias string) (*TxFeed, error) {
	/*	var q bytes.Buffer

		q.WriteString(`
			SELECT id, alias, filter, after
			FROM txfeeds
			WHERE
		`)

		if id != "" {
			q.WriteString(`id=$1`)
		} else {
			q.WriteString(`alias=$1`)
			id = alias
		}

		var (
			feed     TxFeed
			sqlAlias sql.NullString
		)

		err := t.DB.QueryRowContext(ctx, q.String(), id).Scan(&feed.ID, &sqlAlias, &feed.Filter, &feed.After)
		if err == sql.ErrNoRows {
			err = errors.Sub(pg.ErrUserInputNotFound, err)
			err = errors.WithDetailf(err, "alias: %s", alias)
			return nil, err
		}
		if err != nil {
			return nil, err
		}

		if sqlAlias.Valid {
			feed.Alias = &sqlAlias.String
		}
	*/
	//	return &feed, nil
	return nil, nil
}

func (t *Tracker) Get(ctx context.Context, alias string) (feed *TxFeed, err error) {
	if alias == "" {
		return nil, errors.WithDetail(ErrEmptyAlias, "get transaction feed with empty alias")
	}

	for i, v := range t.TxFeeds {
		if v.Alias == alias {
			return &t.TxFeeds[i], nil
		}
	}
	return nil, nil
}

func (t *Tracker) Delete(ctx context.Context, alias string) error {
	log.WithField("delete", alias).Info("delete txfeed")

	if alias == "" {
		return errors.WithDetail(ErrEmptyAlias, "del transaction feed with empty alias")
	}

	for i, txfeed := range t.TxFeeds {
		if txfeed.Alias == alias {
			t.TxFeeds = append(t.TxFeeds[:i], t.TxFeeds[i+1:]...)
			err := deleteTxFeed(t.DB, alias)
			if err != nil {
				return err
			}

			return nil
		}
	}
	return nil
}

func (t *Tracker) Update(ctx context.Context, id, alias, after, prev string) (*TxFeed, error) {
	/*	var q bytes.Buffer

		q.WriteString(`UPDATE txfeeds SET after=$1 WHERE `)

		if id != "" {
			q.WriteString(`id=$2`)
		} else {
			q.WriteString(`alias=$2`)
			id = alias
		}

		q.WriteString(` AND after=$3`)

		res, err := t.DB.ExecContext(ctx, q.String(), after, id, prev)
		if err != nil {
			return nil, err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}

		if affected == 0 {
			return nil, errors.WithDetailf(pg.ErrUserInputNotFound, "could not find txfeed with id/alias=%s and prev=%s", id, prev)
		}

		return &TxFeed{
			ID:    id,
			Alias: &alias,
			After: after,
		}, nil
	*/
	/*	return &TxFeed{
			ID:	nil,
			Alias	nil,
			After	nil,
		}
	*/return nil, nil
}

//TxFilter filter tx from mempool
func (t *Tracker) TxFilter(tx *legacy.Tx) error {
	var annotatedTx *query.AnnotatedTx
	// Build the fully annotated transaction.
	annotatedTx = buildAnnotatedTransaction(tx)
	for _, value := range annotatedTx.Outputs {
		assetid, _ := json.Marshal(value.AssetID)
		assetidstr := string(assetid)
		assetidstr = strings.Replace(assetidstr, "\"", "", -1)
		for _, txfeed := range t.TxFeeds {
			if 0 == strings.Compare(txfeed.Param.assetID, assetidstr) || txfeed.Param.assetID == "" {
				if 0 == strings.Compare(txfeed.Param.transType, value.Type) || txfeed.Param.transType == "" {
					if txfeed.Param.amountLowerLimit < value.Amount || txfeed.Param.amountLowerLimit == 0 {
						if txfeed.Param.amountUpperLimit > value.Amount || txfeed.Param.amountUpperLimit == 0 {
							localAnnotator(annotatedTx)
							b, err := json.Marshal(annotatedTx)
							if err != nil {
								return err
							}
							log.WithField("filter", string(b)).Info("find new tx match filter")

							t.txfeedCh <- tx
							return nil
						}

					}

				}

			}
		}
	}
	return nil
}

var emptyJSONObject = json.RawMessage(`{}`)

func buildAnnotatedTransaction(orig *legacy.Tx) *query.AnnotatedTx {
	tx := &query.AnnotatedTx{
		ID:            orig.ID,
		ReferenceData: &emptyJSONObject,
		Inputs:        make([]*query.AnnotatedInput, 0, len(orig.Inputs)),
		Outputs:       make([]*query.AnnotatedOutput, 0, len(orig.Outputs)),
	}
	/*if pg.IsValidJSONB(orig.ReferenceData) {
		referenceData := json.RawMessage(orig.ReferenceData)
		tx.ReferenceData = &referenceData
	}*/
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
		AssetTags:       &emptyJSONObject,
		ReferenceData:   &emptyJSONObject,
	}
	/*if pg.IsValidJSONB(orig.ReferenceData) {
		referenceData := json.RawMessage(orig.ReferenceData)
		in.ReferenceData = &referenceData
	}*/

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
		AssetTags:       &emptyJSONObject,
		Amount:          orig.Amount,
		ControlProgram:  orig.ControlProgram,
		ReferenceData:   &emptyJSONObject,
	}
	/*if pg.IsValidJSONB(orig.ReferenceData) {
		referenceData := json.RawMessage(orig.ReferenceData)
		out.ReferenceData = &referenceData
	}*/
	if vmutil.IsUnspendable(out.ControlProgram) {
		out.Type = "retire"
	} else {
		out.Type = "control"
	}
	return out
}

// localAnnotator depends on the asset and account annotators and
// must be run after them.
func localAnnotator(tx *query.AnnotatedTx) {
	for _, in := range tx.Inputs {
		if in.AccountID != "" {
			tx.IsLocal, in.IsLocal = true, true
		}
		if in.Type == "issue" && in.AssetIsLocal {
			tx.IsLocal, in.IsLocal = true, true
		}
	}

	for _, out := range tx.Outputs {
		if out.AccountID != "" {
			tx.IsLocal, out.IsLocal = true, true
		}
	}
}
