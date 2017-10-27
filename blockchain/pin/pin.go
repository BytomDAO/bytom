package pin

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/bytom/errors"
	"github.com/bytom/log"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc/legacy"

	dbm "github.com/tendermint/tmlibs/db"
)

const processorWorkers = 10

type Store struct {
	DB dbm.DB

	mu   sync.Mutex
	cond sync.Cond
	pins map[string]*pin
}

func NewStore(db dbm.DB) *Store {
	s := &Store{
		DB:   db,
		pins: make(map[string]*pin),
	}
	s.cond.L = &s.mu
	return s
}

func (s *Store) ProcessBlocks(ctx context.Context, c *protocol.Chain, pinName string, cb func(context.Context, *legacy.Block) error) {
	p := <-s.pin(pinName)
	height := p.getHeight()
	for {
		select {
		case <-ctx.Done():
			log.Error(ctx, ctx.Err())
			return
		case <-c.BlockWaiter(height + 1):
			select {
			case <-ctx.Done():
				log.Error(ctx, ctx.Err())
				return
			case p.sem <- true:
				go p.processBlock(ctx, c, height+1, cb)
				height++
			}
		}
	}
}

func (s *Store) CreatePin(ctx context.Context, name string, height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pins[name]; ok {
		return nil
	}

	block_processor, err := json.Marshal(&struct {
		Name   string
		Height uint64
	}{Name: name,
		Height: height})
	if err != nil {
		return errors.Wrap(err, "failed marshal block_processor")
	}
	if len(block_processor) > 0 {
		s.DB.Set(json.RawMessage("blp"+name), block_processor)
	}

	s.pins[name] = newPin(s.DB, name, height)
	s.cond.Broadcast()
	return nil
}

func (s *Store) Height(name string) uint64 {
	p := <-s.pin(name)
	return p.getHeight()
}

func (s *Store) LoadAll(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var block_processor = struct {
		Name   string
		Height uint64
	}{}
	iter := s.DB.Iterator()
	for iter.Next() {
		key := string(iter.Key())
		if key[:3] != "blp" {
			continue
		}
		err := json.Unmarshal(iter.Value(), &block_processor)
		if err != nil {
			return errors.New("failed unmarshal this block_processor.")
		}

		s.pins[block_processor.Name] = newPin(s.DB,
			block_processor.Name,
			block_processor.Height)

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

type pin struct {
	mu        sync.Mutex
	cond      sync.Cond
	height    uint64
	completed []uint64

	db   dbm.DB
	name string
	sem  chan bool
}

func newPin(db dbm.DB, name string, height uint64) *pin {
	p := &pin{db: db, name: name, height: height, sem: make(chan bool, processorWorkers)}
	p.cond.L = &p.mu
	return p
}

func (p *pin) getHeight() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.height
}

func (p *pin) processBlock(ctx context.Context, c *protocol.Chain, height uint64, cb func(context.Context, *legacy.Block) error) {
	defer func() { <-p.sem }()
	for {
		block, err := c.GetBlock(height)
		if err != nil {
			log.Error(ctx, err)
			continue
		}

		err = cb(ctx, block)
		if err != nil {
			log.Error(ctx, errors.Wrapf(err, "pin %q callback", p.name))
			continue
		}

		err = p.complete(ctx, block.Height)
		if err != nil {
			log.Error(ctx, err)
		}
		break
	}
}

func (p *pin) complete(ctx context.Context, height uint64) error {
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
		block_processor = struct {
			Name   string
			Height uint64
		}{}
		err error
	)

	bytes := p.db.Get(json.RawMessage("blp" + p.name))
	if bytes != nil {
		err = json.Unmarshal(bytes, &block_processor)
		if err == nil && block_processor.Height >= max {
			goto Noupdate
		}
	}

	block_processor.Name = p.name
	block_processor.Height = max

	bytes, err = json.Marshal(&block_processor)
	if err != nil {
		goto Noupdate
	}
	if len(bytes) > 0 {
		p.db.Set(json.RawMessage("blp"+p.name), bytes)
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
