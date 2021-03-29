package integration

/*
import (
	"encoding/hex"
	"fmt"
	"github.com/bytom/bytom/event"
	"github.com/bytom/bytom/protocol"
	"github.com/bytom/bytom/protocol/state"
	"os"
	"testing"

	"github.com/bytom/bytom/consensus/difficulty"
	"github.com/bytom/bytom/errors"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/bc/types"
)

func TestPoW(t *testing.T) {
	block := blockMap[3][1].block
	s := "9e6291970cb44dd94008c79bcaf9d86f18b4b49ba5b2a04781db7199ed3b9e4e"
	SolveBlock(s, block)
}

func SolveBlock(s string, block *types.Block) error {
	bytes, _ := hex.DecodeString(s)
	var bs [32]byte
	copy(bs[:], bytes)
	seed := bc.NewHash(bs)

	maxNonce := ^uint64(0) // 2^64 - 1
	header := &block.BlockHeader
	for i := uint64(0); i < maxNonce; i++ {
		header.Nonce = i
		headerHash := header.Hash()
		if difficulty.CheckProofOfWork(&headerHash, &seed, header.Bits) {
			fmt.Printf("nonce:%v, headerHash:%s \n", header.Nonce, headerHash.String())
			return nil
		}
	}

	return errors.New("not found nonce")
}

func TestHash(t *testing.T) {
	s := "dcaafb317d6faee190410e0c9b99b8e2ac84e748188e54a48c6569890f83ff38"
	bytes, _ := hex.DecodeString(s)

	var bs [32]byte
	copy(bs[:], bytes)
	h := bc.NewHash(bs)
	fmt.Println("newHash:", h.String())
	fmt.Println("oldHash:", s)
}

func TestBits(t *testing.T) {
	p := &processBlockTestCase{
		desc:      "attach a block normally",
		newBlock:  blockMap[1][0].block,
		wantStore: createStoreItems([]int{0, 1}, []*attachBlock{blockMap[0][0], blockMap[1][0]}),
		wantBlockIndex: state.NewBlockIndexWithData(
			map[bc.Hash]*state.BlockNode{
				blockMap[0][0].block.Hash(): mustCreateBlockNode(&blockMap[0][0].block.BlockHeader),
				blockMap[1][0].block.Hash(): mustCreateBlockNode(&blockMap[1][0].block.BlockHeader, &blockMap[0][0].block.BlockHeader),
			},
			[]*state.BlockNode{
				mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil),
				mustNewBlockNode(&blockMap[1][0].block.BlockHeader, mustNewBlockNode(&blockMap[0][0].block.BlockHeader, nil)),
			},
		),
		wantOrphanManage: protocol.NewOrphanManage(),
	}

	defer os.RemoveAll(dbDir)
	if p.initStore == nil {
		p.initStore = make([]*storeItem, 0)
	}
	store, db, err := initStore(p)
	if err != nil {
		t.Fatal(err)
	}

	orphanManage := p.initOrphanManage
	if orphanManage == nil {
		orphanManage = protocol.NewOrphanManage()
	}

	txPool := protocol.NewTxPool(store, event.NewDispatcher())
	chain, err := protocol.NewChainWithOrphanManage(store, txPool, orphanManage)
	if err != nil {
		t.Fatal(err)
	}

	isOrphan, err := chain.ProcessBlock(p.newBlock)
	if p.wantError != (err != nil) {
		t.Fatalf("#case(%s) want error:%t, got error:%t", p.desc, p.wantError, err != nil)
	}

	if isOrphan != p.wantIsOrphan {
		t.Fatalf("#case(%s) want orphan:%t, got orphan:%t", p.desc, p.wantIsOrphan, isOrphan)
	}

	if p.wantStore != nil {
		gotStoreItems, err := loadStoreItems(db)
		if err != nil {
			t.Fatal(err)
		}

		if !storeItems(gotStoreItems).equals(p.wantStore) {
			t.Fatalf("#case(%s) want store:%v, got store:%v", p.desc, p.wantStore, gotStoreItems)
		}
	}

	if p.wantBlockIndex != nil {
		blockIndex := chain.GetBlockIndex()
		if !blockIndex.Equals(p.wantBlockIndex) {
			t.Fatalf("#case(%s) want block index:%v, got block index:%v", p.desc, *p.wantBlockIndex, *blockIndex)
		}
	}

	if p.wantOrphanManage != nil {
		if !orphanManage.Equals(p.wantOrphanManage) {
			t.Fatalf("#case(%s) want orphan manage:%v, got orphan manage:%v", p.desc, *p.wantOrphanManage, *orphanManage)
		}
	}
}

func TestPrintBlockMap(t *testing.T) {
	for height := 0; height < 4; height++ {
		blocks := blockMap[height]
		for i, block := range blocks {
			hash := block.block.Hash()
			fmt.Printf("height:%d,index:%d,hash:%s \n", height, i, hash.String())
		}
	}
}
*/
