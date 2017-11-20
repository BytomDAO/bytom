package signers

import (
	"encoding/binary"
	"sync/atomic"
	"time"

	"github.com/bytom/encoding/base32"
)

//1<seq_id ,increase by 1
var seqID uint32 = 0

func nextSeqId() uint32 {

	atomic.AddUint32(&seqID, 1)

	return seqID
}

// see the SQL function next_cahin_id in schema.sql on https://github.com/chain/chain
func IdGenerate() (string, uint64) {
	var ourEpochMS uint64 = 1496635208000
	var n uint64

	nowMS := uint64(time.Now().UnixNano() / 1e6)
	seqIndex := uint64(nextSeqId())
	seqID := uint64(seqIndex % 1024)
	shardID := uint64(5)

	n = (nowMS - ourEpochMS) << 23
	n = n | (shardID << 10)
	n = n | seqID

	bin := make([]byte, 8)
	binary.BigEndian.PutUint64(bin, n)
	encodeString := base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(bin)

	return encodeString, seqIndex

}
