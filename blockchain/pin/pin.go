package pin

import (
	"encoding/json"
	"sort"
	"sync"

	log "github.com/sirupsen/logrus"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/bytom/errors"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/legacy"
)

const (
	processorWorkers   = 10
	blockProcessPreFix = "BLP:"
	walletPreFix       = "WAL:"
)

func blockProcessKey(name string) []byte {
	return []byte(blockProcessPreFix + name)
}

func walletKey(name string) []byte {
	return []byte(walletPreFix + name)
}

type Processor struct {
	Name   string
	Height uint64
}

type WalletInfo struct {
	Height uint64
	Hash   bc.Hash
}

type Store struct {
	DB dbm.DB

	mu          sync.Mutex
	cond        sync.Cond
	pins        map[string]*pin
	AllContinue chan struct{}
}

func NewStore(db dbm.DB) *Store {
	s := &Store{
		DB:          db,
		pins:        make(map[string]*pin),
		AllContinue: make(chan struct{}, 1),
	}
	s.cond.L = &s.mu
	return s
}

func (s *Store) WalletUpdate(c *protocol.Chain, reverse func(*Store, *dbm.Batch, *legacy.Block)) {
	var wallet WalletInfo
	var err error
	var block *legacy.Block
	var sendStop bool

	storeBatch := s.DB.NewBatch()

	if wallet, err = s.GetWalletInfo(); err != nil {
		log.WithField("", err).Warn("get wallet info")
		return
	}

LOOP:

	for !c.InMainChain(wallet.Height, wallet.Hash) {
		if block, err = c.GetBlockByHash(&wallet.Hash); err != nil {
			log.WithField("", err).Error("get block by hash")
			return
		}

		//have a rollback operation,then send a signal for producer to stop new block process
		if !sendStop {
			<-s.AllPinStopper()
			sendStop = true
		}

		//Reverse this block
		reverse(s, &storeBatch, block)
		log.WithField("Height", wallet.Height).Info("start rollback this block")

		wallet.Height = block.Height - 1
		wallet.Hash = block.PreviousBlockHash

	}

	//if true ,means rollback
	if sendStop {
		var blockProcess Processor
		var rawBlockProcess []byte

		blockProIter := s.DB.IteratorPrefix([]byte(blockProcessPreFix))
		for blockProIter.Next() {
			if err = json.Unmarshal(blockProIter.Value(), &blockProcess); err != nil {
				log.WithField("", err).Error("get block processor")
				return
			}

			if blockProcess.Height > wallet.Height {
				blockProcess.Height = wallet.Height
			}

			if rawBlockProcess, err = json.Marshal(&blockProcess); err != nil {
				log.WithField("", err).Error("save block processor")
				return
			}

			//update block processor to db
			storeBatch.Set(blockProIter.Key(), rawBlockProcess)

			log.WithFields(log.Fields{"name": blockProcess.Name,
				"height": blockProcess.Height}).Info("update block processor")

		}
		//release
		blockProIter.Release()
	}

	rawWallet, err := json.Marshal(wallet)
	if err != nil {
		log.WithField("", err).Error("save wallet info")
		return
	}
	//update wallet to db
	storeBatch.Set(walletKey("wallet"), rawWallet)

	//commit to db
	storeBatch.Write()

	if sendStop {
		//update block processor to memory
		for _, pin := range s.pins {
			pin.setHeight(wallet.Height)
		}

		//all block processor continue produce new process
		s.AllContinue <- struct{}{}

		//complete rollback , for next
		sendStop = false
		log.WithField("Height", wallet.Height).Info("success rollback to this block")
	}

	block, _ = c.GetBlockByHeight(wallet.Height + 1)
	//if we already handled the tail of the chain, we wait
	if block == nil {
		<-c.BlockWaiter(wallet.Height + 1)
		if block, err = c.GetBlockByHeight(wallet.Height + 1); err != nil {
			log.WithField("", err).Error("wallet get block by height")
			return
		}
	}

	//if false, means that rollback operation is necessary,then goto LOOP
	if block.PreviousBlockHash == wallet.Hash {
		//next loop will save
		wallet.Height = block.Height
		wallet.Hash = block.Hash()
	}

	//goto next loop
	goto LOOP

}

func (s *Store) GetWalletInfo() (WalletInfo, error) {
	var w WalletInfo
	var rawWallet []byte

	if rawWallet = s.DB.Get(walletKey("wallet")); rawWallet == nil {
		return w, nil
	}

	if err := json.Unmarshal(rawWallet, &w); err != nil {
		return w, err
	}

	return w, nil

}

func (s *Store) ProcessBlocks(c *protocol.Chain, pinName string, cb func(*legacy.Block) error) {
	p := <-s.pin(pinName)
	height := p.getHeight()
	for {
		select {
		case <-p.producerStop:
			log.Warn("Process blocks, received stop signal")
			return
		case <-c.BlockWaiter(height + 1):
			select {
			case <-p.producerStop:
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
		s.DB.Set(blockProcessKey(name), blockProcessor)
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

	blockProIter := s.DB.IteratorPrefix([]byte(blockProcessPreFix))
	defer blockProIter.Release()
	for blockProIter.Next() {

		err := json.Unmarshal(blockProIter.Value(), &blockProcessor)
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

func (s *Store) AllPinStopper() <-chan struct{} {
	ch := make(chan struct{}, 1)

	//send creator stop signal
	for _, pin := range s.pins {
		pin.producerStop <- struct{}{}
	}

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
	mu           sync.Mutex
	cond         sync.Cond
	stop         pinStop
	producerStop chan struct{}
	height       uint64
	completed    []uint64

	db   dbm.DB
	name string
	sem  chan bool
}

func newPin(db dbm.DB, name string, height uint64) *pin {
	p := &pin{db: db, name: name, height: height, sem: make(chan bool, processorWorkers)}
	p.cond.L = &p.mu
	p.stop.cond.L = &p.stop.mu
	p.producerStop = make(chan struct{}, 1)
	return p
}

func (p *pin) getHeight() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.height
}

func (p *pin) setHeight(height uint64) {
	p.mu.Lock()
	p.height = height
	p.mu.Unlock()
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

	// for handle orphan block rollback
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
		err            error
	)

	bytes := p.db.Get(blockProcessKey(p.name))
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
		p.db.Set(blockProcessKey(p.name), bytes)
	}

Noupdate:
	p.completed = p.completed[i:]
	p.height = max
	p.cond.Broadcast()

	return nil
}

type uint64s []uint64

func (a uint64s) Len() int           { return len(a) }
func (a uint64s) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a uint64s) Less(i, j int) bool { return a[i] < a[j] }
