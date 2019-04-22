package mdns

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"net"
	"reflect"
	"time"
)

var wantEvents = []LANPeerEvent{
	{IP: []net.IP{net.IPv4(1, 2, 3, 4)}, Port: 1024},
	{IP: []net.IP{net.IPv4(1, 2, 3, 4), net.IPv4(5, 6, 7, 8)}, Port: 1024},
}

type mockProtocol struct {
	entries chan *LANPeerEvent
}

func newMockProtocol() *mockProtocol {
	return &mockProtocol{
		entries: make(chan *LANPeerEvent, 1024),
	}
}
func (m *mockProtocol) registerService(port int) error {
	return nil
}

func (m *mockProtocol) registerResolver(event chan LANPeerEvent) error {
	for _, peerEvent := range wantEvents {
		event <- peerEvent
	}
	return nil
}

func (m *mockProtocol) stopService() {

}

func (m *mockProtocol) stopResolver() {

}

func TestLanDiscover(t *testing.T) {
	lanDiscv := NewLANDiscover(newMockProtocol(), 12345)
	defer lanDiscv.Stop()

	lanPeerMsgSub, err := lanDiscv.Subscribe()
	if err != nil {
		t.Fatal("subscribe lan peer msg err")
	}

	var gotevents = []LANPeerEvent{}
	timeout := time.After(1 * time.Second)
	for {
		select {
		case obj, ok := <-lanPeerMsgSub.Chan():
			if !ok {
				t.Fatal("subscription channel closed")
				return
			}

			ev, ok := obj.Data.(LANPeerEvent)
			if !ok {
				t.Fatal("event type error")
				continue
			}
			gotevents = append(gotevents, ev)
		case <-timeout:
			if !reflect.DeepEqual(gotevents, wantEvents) {
				t.Fatalf("mismatch for test lan discover got %s want %s", spew.Sdump(gotevents), spew.Sdump(wantEvents))
			}
			return
		}
	}
}
