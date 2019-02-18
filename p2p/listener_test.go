// +build !network

package p2p

import (
	"bytes"
	"testing"
)

func TestListener(t *testing.T) {
	// Create a listener
	l, _ := NewDefaultListener("tcp", "localhost:8001", true)

	// Dial the listener
	lAddr := l.InternalAddress()
	connOut, err := lAddr.Dial()
	if err != nil {
		t.Fatalf("Could not connect to listener address %v", lAddr)
	}

	connIn, ok := <-l.Connections()
	if !ok {
		t.Fatalf("Could not get inbound connection from listener")
	}

	msg := []byte("hi!")
	go connIn.Write(msg)
	b := make([]byte, 32)
	n, err := connOut.Read(b)
	if err != nil {
		t.Fatalf("Error reading off connection: %v", err)
	}

	b = b[:n]
	if !bytes.Equal(msg, b) {
		t.Fatalf("Got %s, expected %s", b, msg)
	}

	// Close the server, no longer needed.
	l.Stop()
}
