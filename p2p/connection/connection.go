package connection

import (
	"bufio"
	"fmt"
	"math"
	"net"
	"runtime/debug"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	wire "github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"
	"github.com/tendermint/tmlibs/flowrate"
)

const (
	packetTypePing           = byte(0x01)
	packetTypePong           = byte(0x02)
	packetTypeMsg            = byte(0x03)
	maxMsgPacketPayloadSize  = 1024
	maxMsgPacketOverheadSize = 10 // It's actually lower but good enough
	maxMsgPacketTotalSize    = maxMsgPacketPayloadSize + maxMsgPacketOverheadSize

	numBatchMsgPackets = 10
	minReadBufferSize  = 1024
	minWriteBufferSize = 65536
	updateState        = 2 * time.Second
	pingTimeout        = 40 * time.Second
	flushThrottle      = 100 * time.Millisecond

	defaultSendQueueCapacity   = 1
	defaultSendRate            = int64(512000) // 500KB/s
	defaultRecvBufferCapacity  = 4096
	defaultRecvMessageCapacity = 22020096      // 21MB
	defaultRecvRate            = int64(512000) // 500KB/s
	defaultSendTimeout         = 10 * time.Second
	logModule                  = "p2pConn"
)

type receiveCbFunc func(chID byte, msgBytes []byte)
type errorCbFunc func(interface{})

// Messages in channels are chopped into smaller msgPackets for multiplexing.
type msgPacket struct {
	ChannelID byte
	EOF       byte // 1 means message ends here.
	Bytes     []byte
}

func (p msgPacket) String() string {
	return fmt.Sprintf("MsgPacket{%X:%X T:%X}", p.ChannelID, p.Bytes, p.EOF)
}

/*
MConnection handles message transmission on multiple abstract communication
`Channel`s.  Each channel has a globally unique byte id.
The byte id and the relative priorities of each `Channel` are configured upon
initialization of the connection.

There are two methods for sending messages:
	func (m MConnection) Send(chID byte, msg interface{}) bool {}
	func (m MConnection) TrySend(chID byte, msg interface{}) bool {}

`Send(chID, msg)` is a blocking call that waits until `msg` is successfully queued
for the channel with the given id byte `chID`, or until the request times out.
The message `msg` is serialized using the `tendermint/wire` submodule's
`WriteBinary()` reflection routine.

`TrySend(chID, msg)` is a nonblocking call that returns false if the channel's
queue is full.

Inbound message bytes are handled with an onReceive callback function.
*/
type MConnection struct {
	cmn.BaseService

	conn        net.Conn
	bufReader   *bufio.Reader
	bufWriter   *bufio.Writer
	sendMonitor *flowrate.Monitor
	recvMonitor *flowrate.Monitor
	send        chan struct{}
	pong        chan struct{}
	channels    []*channel
	channelsIdx map[byte]*channel
	onReceive   receiveCbFunc
	onError     errorCbFunc
	errored     uint32
	config      *MConnConfig

	quit         chan struct{}
	flushTimer   *cmn.ThrottleTimer // flush writes as necessary but throttled.
	pingTimer    *time.Ticker       // send pings periodically
	chStatsTimer *time.Ticker       // update channel stats periodically
}

// MConnConfig is a MConnection configuration.
type MConnConfig struct {
	SendRate int64 `mapstructure:"send_rate"`
	RecvRate int64 `mapstructure:"recv_rate"`
}

// DefaultMConnConfig returns the default config.
func DefaultMConnConfig() *MConnConfig {
	return &MConnConfig{
		SendRate: defaultSendRate,
		RecvRate: defaultRecvRate,
	}
}

// NewMConnectionWithConfig wraps net.Conn and creates multiplex connection with a config
func NewMConnectionWithConfig(conn net.Conn, chDescs []*ChannelDescriptor, onReceive receiveCbFunc, onError errorCbFunc, config *MConnConfig) *MConnection {
	mconn := &MConnection{
		conn:        conn,
		bufReader:   bufio.NewReaderSize(conn, minReadBufferSize),
		bufWriter:   bufio.NewWriterSize(conn, minWriteBufferSize),
		sendMonitor: flowrate.New(0, 0),
		recvMonitor: flowrate.New(0, 0),
		send:        make(chan struct{}, 1),
		pong:        make(chan struct{}, 1),
		channelsIdx: map[byte]*channel{},
		channels:    []*channel{},
		onReceive:   onReceive,
		onError:     onError,
		config:      config,

		pingTimer:    time.NewTicker(pingTimeout),
		chStatsTimer: time.NewTicker(updateState),
	}

	for _, desc := range chDescs {
		descCopy := *desc // copy the desc else unsafe access across connections
		channel := newChannel(mconn, &descCopy)
		mconn.channelsIdx[channel.id] = channel
		mconn.channels = append(mconn.channels, channel)
	}
	mconn.BaseService = *cmn.NewBaseService(nil, "MConnection", mconn)
	return mconn
}

// OnStart implements BaseService
func (c *MConnection) OnStart() error {
	c.BaseService.OnStart()
	c.quit = make(chan struct{})
	c.flushTimer = cmn.NewThrottleTimer("flush", flushThrottle)
	go c.sendRoutine()
	go c.recvRoutine()
	return nil
}

// OnStop implements BaseService
func (c *MConnection) OnStop() {
	c.BaseService.OnStop()
	c.flushTimer.Stop()
	c.pingTimer.Stop()
	c.chStatsTimer.Stop()
	if c.quit != nil {
		close(c.quit)
	}
	c.conn.Close()
	// We can't close pong safely here because recvRoutine may write to it after we've
	// stopped. Though it doesn't need to get closed at all, we close it @ recvRoutine.
}

// CanSend returns true if you can send more data onto the chID, false otherwise
func (c *MConnection) CanSend(chID byte) bool {
	if !c.IsRunning() {
		return false
	}

	channel, ok := c.channelsIdx[chID]
	if !ok {
		return false
	}
	return channel.canSend()
}

// Send will queues a message to be sent to channel(blocking).
func (c *MConnection) Send(chID byte, msg interface{}) bool {
	if !c.IsRunning() {
		return false
	}

	channel, ok := c.channelsIdx[chID]
	if !ok {
		log.WithFields(log.Fields{"module": logModule, "chID": chID}).Error("cannot send bytes due to unknown channel")
		return false
	}

	if !channel.sendBytes(wire.BinaryBytes(msg)) {
		log.WithFields(log.Fields{"module": logModule, "chID": chID, "conn": c, "msg": msg}).Error("MConnection send failed")
		return false
	}

	select {
	case c.send <- struct{}{}:
	default:
	}
	return true
}

// TrafficStatus return the in and out traffic status
func (c *MConnection) TrafficStatus() (*flowrate.Status, *flowrate.Status) {
	sentStatus := c.sendMonitor.Status()
	receivedStatus := c.recvMonitor.Status()
	return &sentStatus, &receivedStatus
}

// TrySend queues a message to be sent to channel(Nonblocking).
func (c *MConnection) TrySend(chID byte, msg interface{}) bool {
	if !c.IsRunning() {
		return false
	}

	channel, ok := c.channelsIdx[chID]
	if !ok {
		log.WithFields(log.Fields{"module": logModule, "chID": chID}).Error("cannot send bytes due to unknown channel")
		return false
	}

	ok = channel.trySendBytes(wire.BinaryBytes(msg))
	if ok {
		select {
		case c.send <- struct{}{}:
		default:
		}
	}
	return ok
}

func (c *MConnection) String() string {
	return fmt.Sprintf("MConn{%v}", c.conn.RemoteAddr())
}

func (c *MConnection) flush() {
	if err := c.bufWriter.Flush(); err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Warn("MConnection flush failed")
	}
}

// Catch panics, usually caused by remote disconnects.
func (c *MConnection) _recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := cmn.StackError{r, stack}
		c.stopForError(err)
	}
}

// recvRoutine reads msgPackets and reconstructs the message using the channels' "recving" buffer.
// After a whole message has been assembled, it's pushed to onReceive().
// Blocks depending on how the connection is throttled.
func (c *MConnection) recvRoutine() {
	defer c._recover()
	defer close(c.pong)

	for {
		// Block until .recvMonitor says we can read.
		c.recvMonitor.Limit(maxMsgPacketTotalSize, atomic.LoadInt64(&c.config.RecvRate), true)

		// Read packet type
		var n int
		var err error
		pktType := wire.ReadByte(c.bufReader, &n, &err)
		c.recvMonitor.Update(int(n))
		if err != nil {
			if c.IsRunning() {
				log.WithFields(log.Fields{"module": logModule, "conn": c, "error": err}).Warn("Connection failed @ recvRoutine (reading byte)")
				c.conn.Close()
				c.stopForError(err)
			}
			return
		}

		// Read more depending on packet type.
		switch pktType {
		case packetTypePing:
			log.WithFields(log.Fields{"module": logModule, "conn": c}).Debug("receive Ping")
			select {
			case c.pong <- struct{}{}:
			default:
			}

		case packetTypePong:
			log.WithFields(log.Fields{"module": logModule, "conn": c}).Debug("receive Pong")

		case packetTypeMsg:
			pkt, n, err := msgPacket{}, int(0), error(nil)
			wire.ReadBinaryPtr(&pkt, c.bufReader, maxMsgPacketTotalSize, &n, &err)
			c.recvMonitor.Update(int(n))
			if err != nil {
				if c.IsRunning() {
					log.WithFields(log.Fields{"module": logModule, "conn": c, "error": err}).Error("failed on recvRoutine")
					c.stopForError(err)
				}
				return
			}

			channel, ok := c.channelsIdx[pkt.ChannelID]
			if !ok || channel == nil {
				cmn.PanicQ(cmn.Fmt("Unknown channel %X", pkt.ChannelID))
			}

			msgBytes, err := channel.recvMsgPacket(pkt)
			if err != nil {
				if c.IsRunning() {
					log.WithFields(log.Fields{"module": logModule, "conn": c, "error": err}).Error("failed on recvRoutine")
					c.stopForError(err)
				}
				return
			}

			if msgBytes != nil {
				c.onReceive(pkt.ChannelID, msgBytes)
			}

		default:
			cmn.PanicSanity(cmn.Fmt("Unknown message type %X", pktType))
		}
	}
}

// Returns true if messages from channels were exhausted.
func (c *MConnection) sendMsgPacket() bool {
	var leastRatio float32 = math.MaxFloat32
	var leastChannel *channel
	for _, channel := range c.channels {
		if !channel.isSendPending() {
			continue
		}
		if ratio := float32(channel.recentlySent) / float32(channel.priority); ratio < leastRatio {
			leastRatio = ratio
			leastChannel = channel
		}
	}
	if leastChannel == nil {
		return true
	}

	n, err := leastChannel.writeMsgPacketTo(c.bufWriter)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "error": err}).Error("failed to write msgPacket")
		c.stopForError(err)
		return true
	}
	c.sendMonitor.Update(int(n))
	c.flushTimer.Set()
	return false
}

// sendRoutine polls for packets to send from channels.
func (c *MConnection) sendRoutine() {
	defer c._recover()

	for {
		var n int
		var err error
		select {
		case <-c.flushTimer.Ch:
			c.flush()
		case <-c.chStatsTimer.C:
			for _, channel := range c.channels {
				channel.updateStats()
			}
		case <-c.pingTimer.C:
			log.WithFields(log.Fields{"module": logModule, "conn": c}).Debug("send Ping")
			wire.WriteByte(packetTypePing, c.bufWriter, &n, &err)
			c.sendMonitor.Update(int(n))
			c.flush()
		case <-c.pong:
			log.WithFields(log.Fields{"module": logModule, "conn": c}).Debug("send Pong")
			wire.WriteByte(packetTypePong, c.bufWriter, &n, &err)
			c.sendMonitor.Update(int(n))
			c.flush()
		case <-c.quit:
			return
		case <-c.send:
			if eof := c.sendSomeMsgPackets(); !eof {
				select {
				case c.send <- struct{}{}:
				default:
				}
			}
		}

		if !c.IsRunning() {
			return
		}
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "conn": c, "error": err}).Warn("Connection failed @ sendRoutine")
			c.stopForError(err)
			return
		}
	}
}

// Returns true if messages from channels were exhausted.
func (c *MConnection) sendSomeMsgPackets() bool {
	// Block until .sendMonitor says we can write.
	// Once we're ready we send more than we asked for,
	// but amortized it should even out.
	c.sendMonitor.Limit(maxMsgPacketTotalSize, atomic.LoadInt64(&c.config.SendRate), true)
	for i := 0; i < numBatchMsgPackets; i++ {
		if c.sendMsgPacket() {
			return true
		}
	}
	return false
}

func (c *MConnection) stopForError(r interface{}) {
	c.Stop()
	if atomic.CompareAndSwapUint32(&c.errored, 0, 1) && c.onError != nil {
		c.onError(r)
	}
}
