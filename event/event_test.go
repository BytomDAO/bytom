package event

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

type testEvent int

func TestSubCloseUnsub(t *testing.T) {
	// the point of this test is **not** to panic
	var mux Dispatcher
	mux.Stop()
	sub, _ := mux.Subscribe(int(0))
	sub.Unsubscribe()
}

func TestSub(t *testing.T) {
	mux := NewDispatcher()
	defer mux.Stop()

	sub, _ := mux.Subscribe(testEvent(0))
	go func() {
		if err := mux.Post(testEvent(5)); err != nil {
			t.Errorf("Post returned unexpected error: %v", err)
		}
	}()
	ev := <-sub.Chan()

	if ev.Data.(testEvent) != testEvent(5) {
		t.Errorf("Got %v (%T), expected event %v (%T)",
			ev, ev, testEvent(5), testEvent(5))
	}
}

func TestMuxErrorAfterStop(t *testing.T) {
	mux := NewDispatcher()
	mux.Stop()

	sub, _ := mux.Subscribe(testEvent(0))
	if _, isopen := <-sub.Chan(); isopen {
		t.Errorf("subscription channel was not closed")
	}
	if err := mux.Post(testEvent(0)); err != ErrMuxClosed {
		t.Errorf("Post error mismatch, got: %s, expected: %s", err, ErrMuxClosed)
	}
}

func TestUnsubscribeUnblockPost(t *testing.T) {
	mux := NewDispatcher()
	defer mux.Stop()

	sub, _ := mux.Subscribe(testEvent(0))
	unblocked := make(chan bool)
	go func() {
		mux.Post(testEvent(5))
		unblocked <- true
	}()

	select {
	case <-unblocked:
		t.Errorf("Post returned before Unsubscribe")
	default:
		sub.Unsubscribe()
		<-unblocked
	}
}

func TestSubscribeDuplicateType(t *testing.T) {
	mux := NewDispatcher()
	if _, err := mux.Subscribe(testEvent(1), testEvent(2)); err != ErrDuplicateSubscribe {
		t.Fatal("Subscribe didn't error for duplicate type")
	}
}

func TestMuxConcurrent(t *testing.T) {
	rand.Seed(time.Now().Unix())
	mux := NewDispatcher()
	defer mux.Stop()

	recv := make(chan int)
	poster := func() {
		for {
			err := mux.Post(testEvent(0))
			if err != nil {
				return
			}
		}
	}
	sub := func(i int) {
		time.Sleep(time.Duration(rand.Intn(99)) * time.Millisecond)
		sub, _ := mux.Subscribe(testEvent(0))
		<-sub.Chan()
		sub.Unsubscribe()
		recv <- i
	}

	go poster()
	go poster()
	go poster()
	nsubs := 1000
	for i := 0; i < nsubs; i++ {
		go sub(i)
	}

	// wait until everyone has been served
	counts := make(map[int]int, nsubs)
	for i := 0; i < nsubs; i++ {
		counts[<-recv]++
	}
	for i, count := range counts {
		if count != 1 {
			t.Errorf("receiver %d called %d times, expected only 1 call", i, count)
		}
	}
}

func emptySubscriber(mux *Dispatcher) {
	s, _ := mux.Subscribe(testEvent(0))
	go func() {
		for range s.Chan() {
		}
	}()
}

func BenchmarkPost1000(b *testing.B) {
	var (
		mux              = NewDispatcher()
		subscribed, done sync.WaitGroup
		nsubs            = 1000
	)
	subscribed.Add(nsubs)
	done.Add(nsubs)
	for i := 0; i < nsubs; i++ {
		go func() {
			s, _ := mux.Subscribe(testEvent(0))
			subscribed.Done()
			for range s.Chan() {
			}
			done.Done()
		}()
	}
	subscribed.Wait()

	// The actual benchmark.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mux.Post(testEvent(0))
	}

	b.StopTimer()
	mux.Stop()
	done.Wait()
}

func BenchmarkPostConcurrent(b *testing.B) {
	var mux = NewDispatcher()
	defer mux.Stop()
	emptySubscriber(mux)
	emptySubscriber(mux)
	emptySubscriber(mux)

	var wg sync.WaitGroup
	poster := func() {
		for i := 0; i < b.N; i++ {
			mux.Post(testEvent(0))
		}
		wg.Done()
	}
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go poster()
	}
	wg.Wait()
}

// for comparison
func BenchmarkChanSend(b *testing.B) {
	c := make(chan interface{})
	closed := make(chan struct{})
	go func() {
		for range c {
		}
	}()

	for i := 0; i < b.N; i++ {
		select {
		case c <- i:
		case <-closed:
		}
	}
}
