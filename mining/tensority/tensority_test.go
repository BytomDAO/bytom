package tensority

import (
	"reflect"
	"testing"

	"github.com/bytom/common/hexutil"
	"github.com/bytom/protocol/bc"
)

// Tests that tensority hash result is correct.
func TestHash(t *testing.T) {
	tests := []struct {
		blockHeader []byte
		seed        []byte
		hash        []byte
	}{
		{
			blockHeader: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			seed:        hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			hash:        hexutil.MustDecode("0xc5d4a19ce842fee409696d14e483f9efe4a7ecc036d1cfeba0199f13f14dc90f"),
		},
		{
			blockHeader: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			seed:        hexutil.MustDecode("0x48dda5bbe9171a6656206ec56c595c5834b6cf38c5fe71bcb44fe43833aee9df"),
			hash:        hexutil.MustDecode("0x26db94efa422d76c402a54eeb61dd5f53282cd3ce1a0ac677e177051edaa98c1"),
		},
		{
			blockHeader: hexutil.MustDecode("0x8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"),
			seed:        hexutil.MustDecode("0x0e3b78d8380844b0f697bb912da7f4d210382c6714194fd16039ef2acd924dcf"),
			hash:        hexutil.MustDecode("0xfecec33669737592f7754b215b20bacefba64d2e4ca1656f85ea1d3dbe162839"),
		},
		{
			blockHeader: hexutil.MustDecode("0x2f014311e0926fa8b3d6e6de2051bf69332123baadfe522b62f4645655859e7a"),
			seed:        hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			hash:        hexutil.MustDecode("0xc1c3cf4c76968e2967f0053c76f2084cc01ed0fe9766428db99c45bedf0cdbe2"),
		},
		{
			blockHeader: hexutil.MustDecode("0xe0e3c43178a126d04871b9c5d0c642e5e08b9679a5f66b821bd9a030eff02ce7"),
			seed:        hexutil.MustDecode("0x6ab21e1301f5752c2fca1b5598f49d3769482e073c1f26e3b8365f405553ea31"),
			hash:        hexutil.MustDecode("0xabbc2cb39638f684235fbc1b3ff107945948c581b6929bae2cd681889ff2d824"),
		},
		{
			blockHeader: hexutil.MustDecode("0xd0dad73fb2dabf3353fda15571b4e5f6ac62ff187b354fadd4840d9ff2f1afdf"),
			seed:        hexutil.MustDecode("0x0737520781345b11b7bd0f843c1bdd9aea81b6da94fd141cc9f2df53ac6744d2"),
			hash:        hexutil.MustDecode("0xe35da54795d82f8549c0e580cbf2e3757ab5ef8fed1bdbe439416c7e6f8df227"),
		},
	}
	for i, tt := range tests {
		var b32hash [32]byte
		var b32seed [32]byte

		copy(b32hash[:], tt.blockHeader)
		copy(b32seed[:], tt.seed)
		bhhash := bc.NewHash(b32hash)
		bhseed := bc.NewHash(b32seed)

		result := Hash(&bhhash, &bhseed).Bytes()

		if !reflect.DeepEqual(result, tt.hash) {
			t.Errorf("hash %d: content mismatch: have %x, correct %x", i, result, tt.hash)
		}
	}
}
