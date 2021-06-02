package connection

import (
	"bytes"
	"crypto/ed25519"
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/bytom/bytom/crypto/ed25519/chainkd"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/ripemd160"

	"github.com/tendermint/go-wire"
	cmn "github.com/tendermint/tmlibs/common"
)

const (
	dataLenSize     = 2 // uint16 to describe the length, is <= dataMaxSize
	dataMaxSize     = 1024
	totalFrameSize  = dataMaxSize + dataLenSize
	sealedFrameSize = totalFrameSize + secretbox.Overhead
	authSigMsgSize  = 100 // fixed size (length prefixed) byte arrays
)

type authSigMessage struct {
	Key []byte
	Sig []byte
}

// SecretConnection implements net.Conn
type SecretConnection struct {
	conn       io.ReadWriteCloser
	recvBuffer []byte
	recvNonce  *[24]byte
	sendNonce  *[24]byte
	remPubKey  ed25519.PublicKey
	shrSecret  *[32]byte // shared secret
}

// MakeSecretConnection performs handshake and returns a new authenticated SecretConnection.
func MakeSecretConnection(conn io.ReadWriteCloser, locPrivKey chainkd.XPrv) (*SecretConnection, error) {
	locPubKey := locPrivKey.XPub().PublicKey()

	// Generate ephemeral keys for perfect forward secrecy.
	locEphPub, locEphPriv := genEphKeys()

	// Write local ephemeral pubkey and receive one too.
	// NOTE: every 32-byte string is accepted as a Curve25519 public key
	// (see DJB's Curve25519 paper: http://cr.yp.to/ecdh/curve25519-20060209.pdf)
	remEphPub, err := shareEphPubKey(conn, locEphPub)
	if err != nil {
		return nil, err
	}

	// Compute common shared secret.
	shrSecret := computeSharedSecret(remEphPub, locEphPriv)

	// Sort by lexical order.
	loEphPub, hiEphPub := sort32(locEphPub, remEphPub)

	// Generate nonces to use for secretbox.
	recvNonce, sendNonce := genNonces(loEphPub, hiEphPub, locEphPub == loEphPub)

	// Generate common challenge to sign.
	challenge := genChallenge(loEphPub, hiEphPub)

	// Construct SecretConnection.
	sc := &SecretConnection{
		conn:       conn,
		recvBuffer: nil,
		recvNonce:  recvNonce,
		sendNonce:  sendNonce,
		shrSecret:  shrSecret,
	}

	// Sign the challenge bytes for authentication.
	locSignature := signChallenge(challenge, locPrivKey)

	// Share (in secret) each other's pubkey & challenge signature
	authSigMsg, err := shareAuthSignature(sc, locPubKey, locSignature)
	if err != nil {
		return nil, err
	}

	remPubKey, remSignature := authSigMsg.Key, authSigMsg.Sig
	if !ed25519.Verify(remPubKey, challenge[:], remSignature) {
		return nil, errors.New("Challenge verification failed")
	}

	sc.remPubKey = remPubKey
	return sc, nil
}

// CONTRACT: data smaller than dataMaxSize is read atomically.
func (sc *SecretConnection) Read(data []byte) (n int, err error) {
	if 0 < len(sc.recvBuffer) {
		n_ := copy(data, sc.recvBuffer)
		sc.recvBuffer = sc.recvBuffer[n_:]
		return
	}

	sealedFrame := make([]byte, sealedFrameSize)
	if _, err = io.ReadFull(sc.conn, sealedFrame); err != nil {
		return
	}

	// decrypt the frame
	frame := make([]byte, totalFrameSize)
	if _, ok := secretbox.Open(frame[:0], sealedFrame, sc.recvNonce, sc.shrSecret); !ok {
		return n, errors.New("Failed to decrypt SecretConnection")
	}

	incr2Nonce(sc.recvNonce)
	chunkLength := binary.BigEndian.Uint16(frame) // read the first two bytes
	if chunkLength > dataMaxSize {
		return 0, errors.New("chunkLength is greater than dataMaxSize")
	}

	chunk := frame[dataLenSize : dataLenSize+chunkLength]
	n = copy(data, chunk)
	sc.recvBuffer = chunk[n:]
	return
}

// RemotePubKey returns authenticated remote pubkey
func (sc *SecretConnection) RemotePubKey() ed25519.PublicKey {
	return sc.remPubKey
}

// Writes encrypted frames of `sealedFrameSize`
// CONTRACT: data smaller than dataMaxSize is read atomically.
func (sc *SecretConnection) Write(data []byte) (n int, err error) {
	for 0 < len(data) {
		var chunk []byte
		frame := make([]byte, totalFrameSize)
		if dataMaxSize < len(data) {
			chunk = data[:dataMaxSize]
			data = data[dataMaxSize:]
		} else {
			chunk = data
			data = nil
		}
		binary.BigEndian.PutUint16(frame, uint16(len(chunk)))
		copy(frame[dataLenSize:], chunk)

		// encrypt the frame
		sealedFrame := make([]byte, sealedFrameSize)
		secretbox.Seal(sealedFrame[:0], frame, sc.sendNonce, sc.shrSecret)
		incr2Nonce(sc.sendNonce)

		if _, err := sc.conn.Write(sealedFrame); err != nil {
			return n, err
		}

		n += len(chunk)
	}
	return
}

// Close implements net.Conn
func (sc *SecretConnection) Close() error { return sc.conn.Close() }

// LocalAddr implements net.Conn
func (sc *SecretConnection) LocalAddr() net.Addr { return sc.conn.(net.Conn).LocalAddr() }

// RemoteAddr implements net.Conn
func (sc *SecretConnection) RemoteAddr() net.Addr { return sc.conn.(net.Conn).RemoteAddr() }

// SetDeadline implements net.Conn
func (sc *SecretConnection) SetDeadline(t time.Time) error { return sc.conn.(net.Conn).SetDeadline(t) }

// SetReadDeadline implements net.Conn
func (sc *SecretConnection) SetReadDeadline(t time.Time) error {
	return sc.conn.(net.Conn).SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn
func (sc *SecretConnection) SetWriteDeadline(t time.Time) error {
	return sc.conn.(net.Conn).SetWriteDeadline(t)
}

func computeSharedSecret(remPubKey, locPrivKey *[32]byte) (shrSecret *[32]byte) {
	shrSecret = new([32]byte)
	box.Precompute(shrSecret, remPubKey, locPrivKey)
	return
}

func genChallenge(loPubKey, hiPubKey *[32]byte) (challenge *[32]byte) {
	return hash32(append(loPubKey[:], hiPubKey[:]...))
}

// increment nonce big-endian by 2 with wraparound.
func incr2Nonce(nonce *[24]byte) {
	incrNonce(nonce)
	incrNonce(nonce)
}

// increment nonce big-endian by 1 with wraparound.
func incrNonce(nonce *[24]byte) {
	for i := 23; 0 <= i; i-- {
		nonce[i]++
		if nonce[i] != 0 {
			return
		}
	}
}

func genEphKeys() (ephPub, ephPriv *[32]byte) {
	var err error
	ephPub, ephPriv, err = box.GenerateKey(crand.Reader)
	if err != nil {
		log.Panic("Could not generate ephemeral keypairs")
	}
	return
}

func genNonces(loPubKey, hiPubKey *[32]byte, locIsLo bool) (*[24]byte, *[24]byte) {
	nonce1 := hash24(append(loPubKey[:], hiPubKey[:]...))
	nonce2 := new([24]byte)
	copy(nonce2[:], nonce1[:])
	nonce2[len(nonce2)-1] ^= 0x01
	if locIsLo {
		return nonce1, nonce2
	}
	return nonce2, nonce1
}

func signChallenge(challenge *[32]byte, locPrivKey chainkd.XPrv) []byte {
	return locPrivKey.Sign(challenge[:])
}

func shareAuthSignature(sc *SecretConnection, pubKey, signature []byte) (*authSigMessage, error) {
	var recvMsg authSigMessage

	wTask := func(i int) (res interface{}, err error, abort bool) {
		msgBytes := wire.BinaryBytes(authSigMessage{pubKey, signature})
		_, err = sc.Write(msgBytes)
		return nil, err, false
	}

	rTask := func(i int) (res interface{}, err error, abort bool) {
		readBuffer := make([]byte, authSigMsgSize)
		_, err = io.ReadFull(sc, readBuffer)
		if err != nil {
			return nil, err, false
		}

		n := int(0) // not used.
		recvMsg = wire.ReadBinary(authSigMessage{}, bytes.NewBuffer(readBuffer), authSigMsgSize, &n, &err).(authSigMessage)
		return nil, err, false
	}

	trs, ok := cmn.Parallel(wTask, rTask)
	if !ok {
		return nil, errors.New("Parallel task run failed")
	}

	for i := 0; i < 2; i++ {
		res, ok := trs.LatestResult(i)
		if !ok {
			return nil, fmt.Errorf("Task %d did not complete", i)
		}

		if res.Error != nil {
			return nil, fmt.Errorf("Task %d should not has error but god %v", i, res.Error)
		}
	}

	return &recvMsg, nil
}

func shareEphPubKey(conn io.ReadWriteCloser, locEphPub *[32]byte) (remEphPub *[32]byte, err error) {
	var err1, err2 error
	cmn.Parallel(
		func(i int) (res interface{}, err error, abort bool) {
			_, err = conn.Write(locEphPub[:])
			return nil, err, false
		},
		func(i int) (res interface{}, err error, abort bool) {
			remEphPub = new([32]byte)
			_, err = io.ReadFull(conn, remEphPub[:])
			return nil, err, false
		},
	)

	// TODO:
	if err1 != nil {
		return nil, err1
	}
	if err2 != nil {
		return nil, err2
	}
	return remEphPub, nil
}

func sort32(foo, bar *[32]byte) (*[32]byte, *[32]byte) {
	if bytes.Compare(foo[:], bar[:]) < 0 {
		return foo, bar
	}
	return bar, foo
}

// sha256
func hash32(input []byte) (res *[32]byte) {
	hasher := sha256.New()
	hasher.Write(input) // does not error
	resSlice := hasher.Sum(nil)
	res = new([32]byte)
	copy(res[:], resSlice)
	return
}

// We only fill in the first 20 bytes with ripemd160
func hash24(input []byte) (res *[24]byte) {
	hasher := ripemd160.New()
	hasher.Write(input) // does not error
	resSlice := hasher.Sum(nil)
	res = new([24]byte)
	copy(res[:], resSlice)
	return
}
