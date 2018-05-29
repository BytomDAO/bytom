package connection

import (
	"io"
	"sync/atomic"
	"time"

	wire "github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"
)

// ChannelDescriptor is the setting of channel
type ChannelDescriptor struct {
	ID                  byte
	Priority            int
	SendQueueCapacity   int
	RecvBufferCapacity  int
	RecvMessageCapacity int
}

// FillDefaults set the channel config if empty
func (chDesc *ChannelDescriptor) FillDefaults() {
	if chDesc.SendQueueCapacity == 0 {
		chDesc.SendQueueCapacity = defaultSendQueueCapacity
	}
	if chDesc.RecvBufferCapacity == 0 {
		chDesc.RecvBufferCapacity = defaultRecvBufferCapacity
	}
	if chDesc.RecvMessageCapacity == 0 {
		chDesc.RecvMessageCapacity = defaultRecvMessageCapacity
	}
}

type channel struct {
	conn          *MConnection
	desc          *ChannelDescriptor
	id            byte
	sendQueue     chan []byte
	sendQueueSize int32 // atomic.
	recving       []byte
	sending       []byte
	priority      int
	recentlySent  int64 // exponential moving average
}

func newChannel(conn *MConnection, desc *ChannelDescriptor) *channel {
	desc.FillDefaults()
	if desc.Priority <= 0 {
		cmn.PanicSanity("Channel default priority must be a postive integer")
	}
	return &channel{
		conn:      conn,
		desc:      desc,
		id:        desc.ID,
		sendQueue: make(chan []byte, desc.SendQueueCapacity),
		recving:   make([]byte, 0, desc.RecvBufferCapacity),
		priority:  desc.Priority,
	}
}

// Goroutine-safe
// Use only as a heuristic.
func (ch *channel) canSend() bool {
	return ch.loadSendQueueSize() < defaultSendQueueCapacity
}

// Returns true if any msgPackets are pending to be sent.
// Call before calling nextMsgPacket()
// Goroutine-safe
func (ch *channel) isSendPending() bool {
	if len(ch.sending) == 0 {
		if len(ch.sendQueue) == 0 {
			return false
		}
		ch.sending = <-ch.sendQueue
	}
	return true
}

// Goroutine-safe
func (ch *channel) loadSendQueueSize() (size int) {
	return int(atomic.LoadInt32(&ch.sendQueueSize))
}

// Creates a new msgPacket to send.
// Not goroutine-safe
func (ch *channel) nextMsgPacket() msgPacket {
	packet := msgPacket{
		ChannelID: byte(ch.id),
		Bytes:     ch.sending[:cmn.MinInt(maxMsgPacketPayloadSize, len(ch.sending))],
	}
	if len(ch.sending) <= maxMsgPacketPayloadSize {
		packet.EOF = byte(0x01)
		ch.sending = nil
		atomic.AddInt32(&ch.sendQueueSize, -1) // decrement sendQueueSize
	} else {
		packet.EOF = byte(0x00)
		ch.sending = ch.sending[cmn.MinInt(maxMsgPacketPayloadSize, len(ch.sending)):]
	}
	return packet
}

// Handles incoming msgPackets. Returns a msg bytes if msg is complete.
// Not goroutine-safe
func (ch *channel) recvMsgPacket(packet msgPacket) ([]byte, error) {
	if ch.desc.RecvMessageCapacity < len(ch.recving)+len(packet.Bytes) {
		return nil, wire.ErrBinaryReadOverflow
	}

	ch.recving = append(ch.recving, packet.Bytes...)
	if packet.EOF == byte(0x01) {
		msgBytes := ch.recving
		ch.recving = ch.recving[:0] // make([]byte, 0, ch.desc.RecvBufferCapacity)
		return msgBytes, nil
	}
	return nil, nil
}

// Queues message to send to this channel.
// Goroutine-safe
// Times out (and returns false) after defaultSendTimeout
func (ch *channel) sendBytes(bytes []byte) bool {
	select {
	case ch.sendQueue <- bytes:
		atomic.AddInt32(&ch.sendQueueSize, 1)
		return true
	case <-time.After(defaultSendTimeout):
		return false
	}
}

// Queues message to send to this channel.
// Nonblocking, returns true if successful.
// Goroutine-safe
func (ch *channel) trySendBytes(bytes []byte) bool {
	select {
	case ch.sendQueue <- bytes:
		atomic.AddInt32(&ch.sendQueueSize, 1)
		return true
	default:
		return false
	}
}

// Writes next msgPacket to w.
// Not goroutine-safe
func (ch *channel) writeMsgPacketTo(w io.Writer) (n int, err error) {
	packet := ch.nextMsgPacket()
	wire.WriteByte(packetTypeMsg, w, &n, &err)
	wire.WriteBinary(packet, w, &n, &err)
	if err == nil {
		ch.recentlySent += int64(n)
	}
	return
}

// Call this periodically to update stats for throttling purposes.
// Not goroutine-safe
func (ch *channel) updateStats() {
	ch.recentlySent = int64(float64(ch.recentlySent) * 0.8)
}
