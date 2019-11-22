// Package event deals with subscriptions to real-time events.
package event

import (
	"errors"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/bytom/protocol/bc/types"
)

const (
	logModule      = "event"
	maxEventChSize = 65536
)

var (
	// ErrMuxClosed is returned when Posting on a closed TypeMux.
	ErrMuxClosed = errors.New("event: mux closed")
	//ErrDuplicateSubscribe is returned when subscribe duplicate type
	ErrDuplicateSubscribe = errors.New("event: subscribe duplicate type")
)

type NewMinedBlockEvent struct{ Block types.Block }

// TypeMuxEvent is a time-tagged notification pushed to subscribers.
type TypeMuxEvent struct {
	Time time.Time
	Data interface{}
}

// A Dispatcher dispatches events to registered receivers. Receivers can be
// registered to handle events of certain type. Any operation
// called after mux is stopped will return ErrMuxClosed.
//
// The zero value is ready to use.
type Dispatcher struct {
	mutex   sync.RWMutex
	subm    map[reflect.Type][]*Subscription
	stopped bool
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		subm: make(map[reflect.Type][]*Subscription),
	}
}

// Subscribe creates a subscription for events of the given types. The
// subscription's channel is closed when it is unsubscribed
// or the mux is closed.
func (d *Dispatcher) Subscribe(types ...interface{}) (*Subscription, error) {
	sub := newSubscription(d)
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.stopped {
		// set the status to closed so that calling Unsubscribe after this
		// call will short circuit.
		sub.closed = true
		close(sub.postC)
		return sub, nil
	}

	for _, t := range types {
		rtyp := reflect.TypeOf(t)
		oldsubs := d.subm[rtyp]
		if find(oldsubs, sub) != -1 {
			log.WithFields(log.Fields{"module": logModule}).Errorf("duplicate type %s in Subscribe", rtyp)
			return nil, ErrDuplicateSubscribe
		}

		subs := make([]*Subscription, len(oldsubs)+1)
		copy(subs, oldsubs)
		subs[len(oldsubs)] = sub
		d.subm[rtyp] = subs
	}
	return sub, nil
}

// Post sends an event to all receivers registered for the given type.
// It returns ErrMuxClosed if the mux has been stopped.
func (d *Dispatcher) Post(ev interface{}) error {
	event := &TypeMuxEvent{
		Time: time.Now(),
		Data: ev,
	}
	rtyp := reflect.TypeOf(ev)
	d.mutex.RLock()
	if d.stopped {
		d.mutex.RUnlock()
		return ErrMuxClosed
	}

	subs := d.subm[rtyp]
	d.mutex.RUnlock()
	for _, sub := range subs {
		sub.deliver(event)
	}
	return nil
}

// Stop closes a mux. The mux can no longer be used.
// Future Post calls will fail with ErrMuxClosed.
// Stop blocks until all current deliveries have finished.
func (d *Dispatcher) Stop() {
	d.mutex.Lock()
	for _, subs := range d.subm {
		for _, sub := range subs {
			sub.closewait()
		}
	}
	d.subm = nil
	d.stopped = true
	d.mutex.Unlock()
}

func (d *Dispatcher) del(s *Subscription) {
	d.mutex.Lock()
	for typ, subs := range d.subm {
		if pos := find(subs, s); pos >= 0 {
			if len(subs) == 1 {
				delete(d.subm, typ)
			} else {
				d.subm[typ] = posdelete(subs, pos)
			}
		}
	}
	d.mutex.Unlock()
}

func find(slice []*Subscription, item *Subscription) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func posdelete(slice []*Subscription, pos int) []*Subscription {
	news := make([]*Subscription, len(slice)-1)
	copy(news[:pos], slice[:pos])
	copy(news[pos:], slice[pos+1:])
	return news
}

// Subscription is a subscription established through TypeMux.
type Subscription struct {
	dispatcher *Dispatcher
	created    time.Time
	closeMu    sync.Mutex
	closing    chan struct{}
	closed     bool

	// these two are the same channel. they are stored separately so
	// postC can be set to nil without affecting the return value of
	// Chan.
	postMu sync.RWMutex
	readC  <-chan *TypeMuxEvent
	postC  chan<- *TypeMuxEvent
}

func newSubscription(dispatcher *Dispatcher) *Subscription {
	c := make(chan *TypeMuxEvent, maxEventChSize)
	return &Subscription{
		dispatcher: dispatcher,
		created:    time.Now(),
		readC:      c,
		postC:      c,
		closing:    make(chan struct{}),
	}
}

func (s *Subscription) Chan() <-chan *TypeMuxEvent {
	return s.readC
}

func (s *Subscription) Unsubscribe() {
	s.dispatcher.del(s)
	s.closewait()
}

func (s *Subscription) Closed() bool {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	return s.closed
}

func (s *Subscription) closewait() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return
	}
	close(s.closing)
	s.closed = true

	s.postMu.Lock()
	close(s.postC)
	s.postC = nil
	s.postMu.Unlock()
}

func (s *Subscription) deliver(event *TypeMuxEvent) {
	// Short circuit delivery if stale event
	if s.created.After(event.Time) {
		return
	}
	// Otherwise deliver the event
	s.postMu.RLock()
	defer s.postMu.RUnlock()

	select {
	case s.postC <- event:
	case <-s.closing:
	default:
		log.WithFields(log.Fields{"module": logModule}).Errorf("deliver event err unread event size %d", len(s.postC))
	}
}
