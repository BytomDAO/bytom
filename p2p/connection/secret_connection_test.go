package connection

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	cmn "github.com/tendermint/tmlibs/common"

	"github.com/bytom/bytom/crypto/ed25519/chainkd"
)

type dummyConn struct {
	*io.PipeReader
	*io.PipeWriter
}

func (drw dummyConn) Close() (err error) {
	err2 := drw.PipeWriter.CloseWithError(io.EOF)
	err1 := drw.PipeReader.Close()
	if err2 != nil {
		return err
	}
	return err1
}

// Each returned ReadWriteCloser is akin to a net.Connection
func makeDummyConnPair() (fooConn, barConn dummyConn) {
	barReader, fooWriter := io.Pipe()
	fooReader, barWriter := io.Pipe()
	return dummyConn{fooReader, fooWriter}, dummyConn{barReader, barWriter}
}

func makeSecretConnPair(tb testing.TB) (fooSecConn, barSecConn *SecretConnection) {
	fooConn, barConn := makeDummyConnPair()
	fooPrvKey, _ := chainkd.NewXPrv(nil)
	fooPubKey := fooPrvKey.XPub()
	barPrvKey, _ := chainkd.NewXPrv(nil)
	barPubKey := barPrvKey.XPub()

	fooSecConnTask := func(i int) (val interface{}, err error, abort bool) {
		fooSecConn, err = MakeSecretConnection(fooConn, fooPrvKey)
		if err != nil {
			return nil, err, false
		}

		remotePubBytes := fooSecConn.RemotePubKey()
		if !bytes.Equal(remotePubBytes[:], barPubKey[:]) {
			return nil, fmt.Errorf("Unexpected fooSecConn.RemotePubKey.  Expected %v, got %v", barPubKey, remotePubBytes), false
		}

		return nil, nil, false
	}

	barSecConnTask := func(i int) (val interface{}, err error, about bool) {
		barSecConn, err = MakeSecretConnection(barConn, barPrvKey)
		if err != nil {
			return nil, err, false
		}

		remotePubBytes := barSecConn.RemotePubKey()
		if !bytes.Equal(remotePubBytes[:], fooPubKey[:]) {
			return nil, fmt.Errorf("Unexpected barSecConn.RemotePubKey.  Expected %v, got %v", fooPubKey, remotePubBytes), false
		}
		return nil, nil, false
	}

	_, ok := cmn.Parallel(fooSecConnTask, barSecConnTask)
	if !ok {
		tb.Errorf("Parallel task run failed")
	}

	return
}

func TestSecretConnectionHandshake(t *testing.T) {
	fooSecConn, barSecConn := makeSecretConnPair(t)
	fooSecConn.Close()
	barSecConn.Close()
}

func TestSecretConnectionReadWrite(t *testing.T) {
	fooConn, barConn := makeDummyConnPair()
	fooWrites, barWrites := []string{}, []string{}
	fooReads, barReads := []string{}, []string{}

	// Pre-generate the things to write (for foo & bar)
	for i := 0; i < 100; i++ {
		fooWrites = append(fooWrites, cmn.RandStr((cmn.RandInt()%(dataMaxSize*5))+1))
		barWrites = append(barWrites, cmn.RandStr((cmn.RandInt()%(dataMaxSize*5))+1))
	}

	// A helper that will run with (fooConn, fooWrites, fooReads) and vice versa
	genNodeRunner := func(nodeConn dummyConn, nodeWrites []string, nodeReads *[]string) func(int) (interface{}, error, bool) {
		return func(i int) (val interface{}, err error, about bool) {
			// Node handshake
			nodePrvKey, _ := chainkd.NewXPrv(nil)
			nodeSecretConn, err := MakeSecretConnection(nodeConn, nodePrvKey)
			if err != nil {
				return nil, err, false
			}

			nodeWriteTask := func(i int) (val interface{}, err error, about bool) {
				// Node writes
				for _, nodeWrite := range nodeWrites {
					n, err := nodeSecretConn.Write([]byte(nodeWrite))
					if err != nil {
						t.Errorf("Failed to write to nodeSecretConn: %v", err)
						return nil, err, false
					}
					if n != len(nodeWrite) {
						t.Errorf("Failed to write all bytes. Expected %v, wrote %v", len(nodeWrite), n)

						return nil, err, false
					}
				}
				nodeConn.PipeWriter.Close()
				return nil, nil, false
			}

			nodeReadsTask := func(i int) (val interface{}, err error, about bool) {
				// Node reads
				defer nodeConn.PipeReader.Close()
				readBuffer := make([]byte, dataMaxSize)
				for {
					n, err := nodeSecretConn.Read(readBuffer)
					if err == io.EOF {
						return nil, nil, false
					} else if err != nil {
						return nil, err, false
					}
					*nodeReads = append(*nodeReads, string(readBuffer[:n]))
				}
			}

			// In parallel, handle reads and writes
			trs, ok := cmn.Parallel(nodeWriteTask, nodeReadsTask)
			if !ok {
				t.Errorf("Parallel task run failed")
			}
			for i := 0; i < 2; i++ {
				res, ok := trs.LatestResult(i)
				if !ok {
					t.Errorf("Task %d did not complete", i)
				}

				if res.Error != nil {
					t.Errorf("Task %d should not has error but god %v", i, res.Error)
				}
			}
			return
		}
	}
	// Run foo & bar in parallel
	cmn.Parallel(
		genNodeRunner(fooConn, fooWrites, &fooReads),
		genNodeRunner(barConn, barWrites, &barReads),
	)

	// A helper to ensure that the writes and reads match.
	// Additionally, small writes (<= dataMaxSize) must be atomically read.
	compareWritesReads := func(writes []string, reads []string) {
		for {
			// Pop next write & corresponding reads
			var read, write string = "", writes[0]
			var readCount = 0
			for _, readChunk := range reads {
				read += readChunk
				readCount++
				if len(write) <= len(read) {
					break
				}
				if len(write) <= dataMaxSize {
					break // atomicity of small writes
				}
			}
			// Compare
			if write != read {
				t.Errorf("Expected to read %X, got %X", write, read)
			}
			// Iterate
			writes = writes[1:]
			reads = reads[readCount:]
			if len(writes) == 0 {
				break
			}
		}
	}

	compareWritesReads(fooWrites, barReads)
	compareWritesReads(barWrites, fooReads)

}

func BenchmarkSecretConnection(b *testing.B) {
	b.StopTimer()
	fooSecConn, barSecConn := makeSecretConnPair(b)
	fooWriteText := cmn.RandStr(dataMaxSize)
	// Consume reads from bar's reader
	go func() {
		readBuffer := make([]byte, dataMaxSize)
		for {
			_, err := barSecConn.Read(readBuffer)
			if err == io.EOF {
				return
			} else if err != nil {
				b.Fatalf("Failed to read from barSecConn: %v", err)
			}
		}
	}()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := fooSecConn.Write([]byte(fooWriteText))
		if err != nil {
			b.Fatalf("Failed to write to fooSecConn: %v", err)
		}
	}
	b.StopTimer()

	fooSecConn.Close()
	//barSecConn.Close() race condition
}
