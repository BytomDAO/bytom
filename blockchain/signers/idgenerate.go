package signers

import (
	"encoding/binary"
	"github.com/bytom/encoding/base32"
	"sync/atomic"
	"time"
)

//1<seq_id ,increase by 1
var seq_id uint32 = 1

func next_seq_id() uint32 {

	atomic.AddUint32(&seq_id, 1)

	return (seq_id)
}

// see the SQL function next_cahin_id in schema.sql on https://github.com/chain/chain
func Idgenerate(prefix string) (string, uint64) {
	var our_epoch_ms uint64 = 1496635208000
	var n uint64

	now_ms := uint64(time.Now().UnixNano() / 1e6)
	seq_index := uint64(next_seq_id())
	seq_id := uint64(seq_index % 1024)
	shard_id := uint64(5)

	n = (now_ms - our_epoch_ms) << 23
	n = n | (shard_id << 10)
	n = n | seq_id

	bin := make([]byte, 8)
	binary.BigEndian.PutUint64(bin, n)
	encodestring := base32.HexEncoding.WithPadding(base32.NoPadding).EncodeToString(bin)

	return prefix + encodestring, seq_index

}
