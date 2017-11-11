package pin

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/util"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	processorWorkers = 10
	snapPoint        = 1024
	snapSum          = 2
	//copy from package account
	InsertUnspentsPinName = "insert-account-unspents"
)

var GlobalPinStore *Store = nil

type Processor struct {
	Name   string
	Height uint64
}

//copy from package account
type AccountUTXOs struct {
	OutputID     []byte
	AssetID      []byte
	Amount       uint64
	AccountID    string
	ProgramIndex uint64
	Program      []byte
	BlockHeight  uint64
	SourceID     []byte
	SourcePos    uint64
	RefData      []byte
	Change       bool
}

type Store struct {
	DB          dbm.DB
	mu          sync.Mutex
	cond        sync.Cond
	pins        map[string]*pin
	Rollback    chan uint64
	AllContinue chan struct{}
}

func NewStore(db dbm.DB) *Store {
	s := &Store{
		DB:   db,
		pins: make(map[string]*pin),
		Rollback:make(chan uint64,1),
		AllContinue:make(chan struct{},1),
	}
	s.cond.L = &s.mu
	return s
}

func (s *Store) ProcessBlocks(c *protocol.Chain, pinName string, cb func(*legacy.Block) error) {
	p := <-s.pin(pinName)
	height := p.getHeight()
	for {
		select {
		case <-c.PinStop:
			log.Warn("Process blocks, received stop signal")
			return
		case <-c.BlockWaiter(height + 1):
			select {
			case <-c.PinStop:
				log.Warn("Process blocks, received stop signal")
				return
			case p.sem <- true:
				go p.processBlock(c, height+1, cb)
				height++
			}
		}
	}
}

func (s *Store) CreatePin(name string, height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pins[name]; ok {
		return nil
	}

	blockProcessor, err := json.Marshal(&Processor{Name: name, Height: height})
	if err != nil {
		return errors.Wrap(err, "failed marshal blockProcessor")
	}
	if len(blockProcessor) > 0 {
		s.DB.Set(json.RawMessage("blp"+name), blockProcessor)
	}

	s.pins[name] = newPin(s.DB, name, height)
	s.cond.Broadcast()
	return nil
}

func (s *Store) Height(name string) uint64 {
	p := <-s.pin(name)
	return p.getHeight()
}

func (s *Store) LoadAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var blockProcessor Processor

	it := s.DB.IteratorPrefix([]byte("blp"))
	defer it.Release()
	for it.Next() {

		err := json.Unmarshal(it.Value(), &blockProcessor)
		if err != nil {
			return errors.New("failed unmarshal this blockProcessor")
		}

		s.pins[blockProcessor.Name] = newPin(s.DB, blockProcessor.Name, blockProcessor.Height)

	}

	s.cond.Broadcast()
	return nil
}

func (s *Store) pin(name string) <-chan *pin {
	ch := make(chan *pin, 1)
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for s.pins[name] == nil {
			s.cond.Wait()
		}
		ch <- s.pins[name]
	}()
	return ch
}

func (s *Store) PinWaiter(pinName string, height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	p := <-s.pin(pinName)
	go func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		for p.height < height {
			p.cond.Wait()
		}
		ch <- struct{}{}
	}()
	return ch
}

func (s *Store) PinStopper(pinName string) <-chan struct{} {
	ch := make(chan struct{}, 1)
	p := <-s.pin(pinName)
	go func() {
		p.stop.mu.Lock()
		defer p.stop.mu.Unlock()
		//len(p.sem) == 0 means there all no active block processors
		for len(p.sem) > 0 {
			p.stop.cond.Wait()
		}
		ch <- struct{}{}
	}()
	return ch
}

func (s *Store) AllPinStopper(c *protocol.Chain) <-chan struct{} {
	ch := make(chan struct{}, 1)

	//send stop signal
	c.PinStop <- struct{}{}

	go func() {
		var pins []string
		s.mu.Lock()
		for name := range s.pins {
			pins = append(pins, name)
		}
		s.mu.Unlock()
		//make sure all active block processors stop
		for _, name := range pins {
			<-s.PinStopper(name)
		}
		ch <- struct{}{}
	}()

	return ch
}

func (s *Store) AllWaiter(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		var pins []string
		s.mu.Lock()
		for name := range s.pins {
			pins = append(pins, name)
		}
		s.mu.Unlock()
		for _, name := range pins {
			<-s.PinWaiter(name, height)
		}
		ch <- struct{}{}
	}()
	return ch
}

type pinStop struct {
	mu   sync.Mutex
	cond sync.Cond
}

type pin struct {
	mu        sync.Mutex
	cond      sync.Cond
	stop      pinStop
	height    uint64
	completed []uint64

	db   dbm.DB
	name string
	sem  chan bool
}

func newPin(db dbm.DB, name string, height uint64) *pin {
	p := &pin{db: db, name: name, height: height, sem: make(chan bool, processorWorkers)}
	p.cond.L = &p.mu
	p.stop.cond.L = &p.stop.mu
	return p
}

func (p *pin) getHeight() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.height
}

func (p *pin) processBlock(c *protocol.Chain, height uint64, cb func(*legacy.Block) error) {

	for {
		block, err := c.GetBlockByHeight(height)
		if err != nil {
			log.WithField("error", err).Error("Process block")
			continue
		}

		err = cb(block)
		if err != nil {
			log.WithField("Pin name", p.name).Error("Pin callback")
			continue
		}

		err = p.complete(block.Height)
		if err != nil {
			log.WithField("error", err).Error("Process block")
		}
		break
	}

	// for deal with orphan block rollback
	p.stop.mu.Lock()
	<-p.sem
	p.stop.mu.Unlock()
	p.stop.cond.Signal()

}

func (p *pin) complete(height uint64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.completed = append(p.completed, height)
	sort.Sort(uint64s(p.completed))

	var (
		max = p.height
		i   int
	)
	for i = 0; i < len(p.completed); i++ {
		if p.completed[i] <= max {
			continue
		} else if p.completed[i] > max+1 {
			break
		}
		max = p.completed[i]
	}

	if max == p.height {
		return nil
	}

	var (
		blockProcessor Processor
		err error
	)

	bytes := p.db.Get(json.RawMessage("blp" + p.name))
	if bytes != nil {
		err = json.Unmarshal(bytes, &blockProcessor)
		if err == nil && blockProcessor.Height >= max {
			goto Noupdate
		}
	}

	blockProcessor.Name = p.name
	blockProcessor.Height = max

	bytes, err = json.Marshal(&blockProcessor)
	if err != nil {
		goto Noupdate
	}
	if len(bytes) > 0 {
		p.db.Set(json.RawMessage("blp"+p.name), bytes)
	}

	if p.name == InsertUnspentsPinName && max > 0 && max%1024 == 0 {
		go GlobalPinStore.StoreSnapshot(max)
	}

Noupdate:
	p.completed = p.completed[i:]
	p.height = max
	p.cond.Broadcast()

	return nil
}

func (s *Store) StoreSnapshot(height uint64) {
	var au = AccountUTXOs{}
	var blockProcessor = Processor{}

	storeBatch := s.DB.NewBatch()
	db, _ := s.DB.(*dbm.GoLevelDB)

	goLevelDB := db.DB()

	newSnapshot, err := goLevelDB.GetSnapshot()
	if err != nil {
		log.WithField("err", err).Error("saving accountutxos snapshot")
		return
	}

	//delete old  snapshot
	oldSnapPoint := (height / snapPoint) - snapSum
	if oldSnapPoint > 0 {
		oldprefix := fmt.Sprintf("snp%d", oldSnapPoint)
		it1 := newSnapshot.NewIterator(util.BytesPrefix([]byte(oldprefix)), nil)
		for it1.Next() {
			storeBatch.Delete(it1.Key())
		}
		it1.Release()
	}

	//save new account unspent outputs snapshot
	// must not have snp0
	newPrefix := fmt.Sprintf("snp%d", height/snapPoint)
	it2 := newSnapshot.NewIterator(util.BytesPrefix([]byte("acu")), nil)
	for it2.Next() {
		err = json.Unmarshal(it2.Value(), &au)
		if err != nil || au.BlockHeight > height {
			log.WithFields(log.Fields{"err": err, "hash": string(au.OutputID)}).Warn("" +
				"saving accountutxos snapshot")
			continue
		}

		storeBatch.Set([]byte(newPrefix+string(it2.Key())), it2.Value())
	}
	it2.Release()

	//save new block processors snapshot
	it3 := newSnapshot.NewIterator(util.BytesPrefix([]byte("blp")), nil)
	for it3.Next() {
		err = json.Unmarshal(it3.Value(), &blockProcessor)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "name": blockProcessor.Name}).Warn("" +
				"saving accountutxos snapshot")
			continue
		}
		storeBatch.Set([]byte(newPrefix+string(it3.Key())), it3.Value())
	}
	it3.Release()

	//commit
	storeBatch.Write()

	newSnapshot.Release()
}

func (s *Store) StoreRollBack(bestHeight uint64) {
	log.WithField("rollback height", bestHeight).Info("account unspent outputs start to rollback")

	rollBackPrefix := ""
	deletePrefix := ""
	rollBackKey := []byte{}
	storeBatch := s.DB.NewBatch()

	if (bestHeight/snapPoint) > 0 && (bestHeight%snapPoint) == 0 {
		rollBackPrefix = fmt.Sprintf("snp%d", (bestHeight/snapPoint)-1)
		deletePrefix = fmt.Sprintf("snp%d", bestHeight/snapPoint)
	} else {
		rollBackPrefix = fmt.Sprintf("snp%d", bestHeight/snapPoint)
	}

	//delete invalid store snapshot
	it0 := s.DB.IteratorPrefix([]byte(deletePrefix))
	for it0.Next() {
		storeBatch.Delete(it0.Key())
	}
	it0.Release()

	// delete old account unspent outputs
	it1 := s.DB.IteratorPrefix([]byte("acu"))
	for it1.Next() {
		storeBatch.Delete(it1.Key())
	}
	it1.Release()

	// delete old block processor
	it2 := s.DB.IteratorPrefix([]byte("blp"))
	for it2.Next() {
		storeBatch.Delete(it2.Key())
	}
	it2.Release()

	// rollback
	it3 := s.DB.IteratorPrefix([]byte(rollBackPrefix))
	for it3.Next() {
		rollBackKey = it3.Key()
		storeBatch.Set(rollBackKey[len(rollBackPrefix):], it3.Value())
	}
	it3.Release()

	//commit
	storeBatch.Write()

	//all block processors continue
	GlobalPinStore.AllContinue <- struct{}{}

	log.Info("account unspent outputs rollback end")
}

func (s *Store) StoreListener(count int) {

	rollBackHeight := <-s.Rollback

	log.WithField("count", count).Info("start new store rollback lister")

	s.StoreRollBack(rollBackHeight)

	// start one new listen and return this listen
	go s.StoreListener(count + 1)

	return
}

type uint64s []uint64

func (a uint64s) Len() int           { return len(a) }
func (a uint64s) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a uint64s) Less(i, j int) bool { return a[i] < a[j] }
