package tensority

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/bytom/common/hexutil"
)

// Tests that calcSeedCache create correct result.
func TestCalcSeedCache(t *testing.T) {
	tests := []struct {
		seed  []byte
		cache []byte
	}{
		{
			seed: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
			cache: hexutil.MustDecode("0x" +
				"00000000a1bc45c16d62c61b2f807b3f0a53e759395c8c7fca7ad4327d2fa02d" +
				"2c11783733e72ee949051ebf7acf67b360b6e8717f62dbcd977e7910feed9ed2" +
				"2c6095043edca5426acb692f038b265326e07554f52ad903c5625e18a2d697b9" +
				"d70c19b85fc24f772d707f3ef1dd8d537aa980b8c3b0c8496fa93aecb35157cd" +
				"644aa5318624401924c2e49f9302e48b56e299e1b2419b35e970f9af47497c24" +
				"06d0ecfc715e5d2b80e0137f0ee774d34126f491ad64089620acf7595622742c" +
				"2a2d9eb8ae286a4fd5aaa144d867b0e02bdc64cb045bd605358e5fd1174fde56" +
				"22cfc4b7f2954ca58df2920e5f9003d3bb9c167a5e9a917739dff71a47f1ca0f" +
				"aa3c8b2a0d38bb2941cff24d2622322177b99e94416a1cb4fab1840f258178bd" +
				"bc8bcbbbc95155dc88730b5259af8bab63aff65663e3bc7c58c95aad47717de0" +
				"28a47ae730037df0e0420d70909300ca122d1e85a40181d983f832749df229df" +
				"67041bf4637a97f91ea0f8ed531803eaac6097c85cbff29f7c53e5d10145cd5b" +
				"4dbfdf50c12db6ef7b73af7f7ec1e7a5f23e9fa71f39bde7b9181607d90f902d" +
				"977f79d80eb395a30504088f28c50aa01fd007ff60b83ab4f38fc8a75f11f5ec" +
				"1b38144cb18e38cb800d9ae6bef35eea6182d8fce7649e1cfbf93c9822d02148" +
				"437af3d192ecd3d9485c455b319a759ab6db87bd6e3cff230e3676d02ec2031f"),
		},
		{
			seed: hexutil.MustDecode("0xebabd1bf80ecd995cb40c3b39f391732d47fefec05440ccd65f1da6566e2b910"),
			cache: hexutil.MustDecode("0x" +
				"ebabd1bf1489eeac4282db59ed233113e837e1a72a64ff725f8a91543c7a86d6" +
				"2b3f50c6aaa5cc4d34cbc6dcf012f5bd42fc353dd1268a7a581a67d15cfe2e45" +
				"b08bbcb0bcc860688c2cdba9475c11418ec96cf70e3551d77d06d97445740978" +
				"0da6ca20f29448fcf910ec865216aa61d2d325dbc1ae7854e5127478e6d91148" +
				"03ea5afaf13efdcb50700a6d79a4ed3b2982e3f673636add24095ad5f9719339" +
				"e3ca9b2fe1a8d15f8f14993a62f47aa28227ebedaff488d836c8ab9d49c95496" +
				"6de69a98eee3308a24892f9b12127015592001c9627bc20e0d18c5c33013782e" +
				"ef9055a4c604f2ed81213736e29c3cbfbe8a6d840b89ca9c1f1aaf6f6ca2e468" +
				"929b46f081d0855941f288fa9c2e35ef3c28bd047714c178cc05dc79ec11275b" +
				"12dd65768f1f14a71756b4153e52108e3028548e65dfd2a9ecd32237296f02c2" +
				"5a9c0991bc12fbaec4239748d221153c739356a7963c4fee1a237f7b66784045" +
				"515d6e200822fa5cc7dd64665d1ed56b9bc0b9fc7b39292b38d7f9b757f84a14" +
				"c6683e3d722493aa34d132cf6bdea8d6a69cc5ec0c5277b78f98565921a6711c" +
				"623e6d47dc3cb6821a21cc5d6a711e0aa920baaeb0a4a2368b2485e093a5a2a2" +
				"8d55343f4948fc3e3425e0b61b243c1b6783c2a78772133ab639c869e05b80a1" +
				"9a9943854442fdb5c1bbf71a0b93fc7fa9859bbcea3d7973b4b73492b6060219"),
		},
		{
			seed: hexutil.MustDecode("0x67e35e59db0e26d0ad4e4cabb4ed0339234e3cfd0c61068ac7369db2502ad43e"),
			cache: hexutil.MustDecode("0x" +
				"67e35e596a169fbe0174daad8a0dc617807565500a3f5dd9c0da6d322adbc3d5" +
				"e5e5fbdb4a64789747038c41b197b91c36e8bf7b954db41c8a9b29d2501b97ae" +
				"2151dc00b0fe9909a3f71db46704888ea7f28733bc1d14544fd169e83e11e500" +
				"e6aa723faf610a9983379bb025e5721c9156e3e1b9762556a47603e02dde175b" +
				"e093639b60b8f50b563b746ceebfc742c1ed485a2f07b6bd3a03ebf60d636e05" +
				"1f7dcf57b6f8904b186d764aad9b8a3829f42f51c368e42317939c402799728e" +
				"316380e615ace16a3fffbfe1739ff9a666c65720925a2c4c82b035bc2a7cd835" +
				"b8a3cd5b53c9a282378daf601b524538cbb07671b46044922856512c38805f6f" +
				"a4916d0e2c007a4cdb78cd5bdc6f2de517edfa610d5fb9ab511eebf5f90d66b3" +
				"0947449939b755b8a39764c6eeded10202a848734ed6154dd6d089cafcaca472" +
				"d96d3a4a62ae94b2c0005f4a91afe9fcf812e5a562ea753541c62ad9685b21c1" +
				"b5e447cd98ec9851d8e74104fc94655bbea6766b814949191f63740608fea4d0" +
				"c4bde5da46ee694b412cbc7aa3362dd8916c3c38bd0fe3280607a920b9ac9427" +
				"7f038f373cc6c6dbe18e565d2b80c4a99fe937d8b54024f3d26b86c0d0039034" +
				"f4e2a58c696f55ede5d5ef7558bf02b9e2faf26aff0961ecd173174d65d41316" +
				"68a144612f789ad4e176224171fb33537a5d511c508232f13767a62b40582662"),
		},
	}
	for i, tt := range tests {
		result := calcSeedCache(tt.seed)

		want := make([]uint32, 128)
		prepare(want, tt.cache)

		for j := 0; j < 128; j++ {
			if !reflect.DeepEqual(result[j*1024*32:j*1024*32+1], want[j:j+1]) {
				t.Errorf("cache %d: content mismatch: have %x, want %x", i, result[j*1024*32:j*1024*32+1], want[j:j+1])
			}
		}
	}
}

// prepare converts an ethash cache or dataset from a byte stream into the internal
// int representation. All ethash methods work with ints to avoid constant byte to
// int conversions as well as to handle both little and big endian systems.
func prepare(dest []uint32, src []byte) {
	for i := 0; i < len(dest); i++ {
		dest[i] = binary.LittleEndian.Uint32(src[i*4:])
	}
}
