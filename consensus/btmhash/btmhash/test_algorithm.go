package btmhash

import (
	//"encoding/binary"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/crypto/sha3"

	"btmhash/matrix"
)

/*
	"7ce2991c951f7bf4c4c1bb119887ee07871eb5339d7b97b8588e85c742de90e5bafd5bbe6ce93a134fb6be9ad3e30db99d9528a2ea7846833f52e9ca119b6b54" +
	"8979480c46e19972bd0738779c932c1b43e665a2fd3122fc3ddb2691f353ceb0ed3e38b8f51fd55b6940290743563c9f8fa8822e611924657501a12aafab8a8d" +
	"88fb5fbae3a99d14792406672e783a06940a42799b1c38bc28715db6d37cb11f9f6b24e386dc52dd8c286bd8c36fa813dffe4448a9f56ebcbeea866b42f68d22" +
	"6c32aae4d695a23cab28fd74af53b0c2efcc180ceaaccc0b2e280103d097a03c1d1b0f0f26ce5f32a90238f9bc49f645db001ef9cd3d13d44743f841fad11a37" +
	"fa290c62c16042f703578921f30b9951465aae2af4a5dad43a7341d7b4a62750954965a47a1c3af638dc3495c4d62a9bab843168c9fc0114e79cffd1b2827b01" +
	"75d30ba054658f214e946cf24c43b40d3383fbb0493408e5c5392434ca21bbcf43200dfb876c713d201813934fa485f48767c5915745cf0986b1dc0f33e57748" +
	"bf483ee2aff4248dfe461ec0504a13628401020fc22638584a8f2f5206a13b2f233898c78359b21c8226024d0a7a93df5eb6c282bdbf005a4aab497e096f2847" +
	"76c71cee57932a8fb89f6d6b8743b60a4ea374899a94a2e0f218d5c55818cefb1790c8529a76dba31ebb0f4592d709b49587d2317970d39c086f18dd244291d9" +
	"eedb16705e53e3350591bd4ff4566a3595ac0f0ce24b5e112a3d033bc51b6fea0a92296dea7f5e20bf6ee6bc347d868fda193c395b9bb147e55e5a9f67cfe741" +
	"7eea7d699b155bd13804204df7ea91fa9249e4474dddf35188f77019c67d201e4c10d7079c5ad492a71afff9a23ca7e900ba7d1bdeaf3270514d8eb35eab8a0a" +
	"718bb7273aeb37768fa589ed8ab01fbf4027f4ebdbbae128d21e485f061c20183a9bc2e31edbda0727442e9d58eb0fe198440fe199e02e77c0f7b99973f1f74c" +
	"c9089a51ab96c94a84d66e6aa48b2d0a4543adb5a789039a2aa7b335ca85c91026c7d3c894da53ae364188c3fd92f78e01d080399884a47385aa792e38150cda" +
	"a8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995c" +
	"778401b94a2e66e6993ad67ad3ecdc2acb17779f1ea8606827ec92b11c728f8c3b6d3f04a3e6ed05ff81dd76d5dc5695a50377bc135aaf1671cf68b750315493" +
	"6c64510164d53312bf3c41740c7a237b05faf4a191bd8a95dafa068dbcf370255c725900ce5c934f36feadcfe55b687c440574c1f06f39d207a8553d39156a24" +
	"845f64fd8324bb85312979dead74f764c9677aab89801ad4f927f1c00f12e28f22422bb44200d1969d9ab377dd6b099dc6dbc3222e9321b2c1e84f8e2f07731c")
*/

/*
	"1f56855d59cc5a085720899b4377a0198f1abe948d85fe5820dc0e346b7c0931b9cde8e541d751de3b2b3275d0aabfae316209d5879297d8bd99f8a033c9d4df" +
	"35add1029f4e6404a022d504fb8023e42989aba985a65933b0109c7218854356f9284983c9e7de97de591828ae348b63d1fc78d8db58157344d4e06530ffd422" +
	"5c7f6080d451ff94961ec2dd9e28e6d81b49102451676dbdcb6ef1094c1e8b29e7e808d47b2ba5aeb52dabf00d5f0ee08c116289cbf56d8132e5ca557c3d6220" +
	"5ba3a48539acabfd4ca3c89e3aaa668e24ffeaeb9eb0136a9fc5a8a676b6d5ad76175eeda0a1fa44b5ff5591079e4b7f581569b6c82416adcb82d7e92980df67" +
	"2248c4024013e7be52cf91a82491627d9e6d80eda2770ab82badc5e120cd33a4c84495f718b57396a8f397e797087fad81fa50f0e2f5da71e40816a85de35a96" +
	"3cd351364905c45b3116ff25851d43a2ca1d2aa5cdb408440dabef8c57778fc18608bf431d0c7ffd37649a21a7bb9d90def39c821669dbaf165c0262434dfb08" +
	"5d057a12de4a7a59fd2dfc931c29c20371abf748b69b618a9bd485b3fb3166cad4d3d27edf0197aabeceb28b96670bdf020f26d1bb9b564aaf82d866bdffd6d4" +
	"1aea89e20b15a5d1264ab01d1556bfc2a266081609d60928216bd9646038f07de9fedcc9f2b86ab1b07d7bd88ba1df08b3d89b2ac789001b48a723f217debcb7" +
	"090303a3ef50c1d5d99a75c640ec2b401ab149e06511753d8c49cafdde2929ae61e09cc0f0319d262869d21ead9e0cf5ff2de3dbedfb994f32432d2e4aa44c82" +
	"7c42781d1477fe03ea0772998e776d63363c6c3edd2d52c89b4d2c9d89cdd90fa33b2b41c8e3f78ef06fe90bcf5cc5756d33a032f16b744141aaa8852bb4cb3a" +
	"40792b93489c6d6e56c235ec4aa36c263e9b766a4daaff34b2ea709f9f811aef498a65bfbc1deffd36fcc4d1a123345fac7bf57a1fb50394843cd28976a6c7ff" +
	"fe70f7b8d8f384aa06e2c9964c92a8788cef397fffdd35181b42a35d5d98cd7244bbd09e802888d7efc0311ae58e0961e3656205df4bdc553f317df4b6ede4ca" +
	"846294a32aec830ab1aa5aac4e78b821c35c70fd752fec353e373bf9be656e775a0111bcbeffdfebd3bd5251d27b9f6971aa561a2bd27a99d61b2ce3965c3726" +
	"1e114353e6a31b09340f4078b8a8c6ce6ff4213067a8f21020f78aff4f8b472b701ef730aacb8ce7806ea31b14abe8f8efdd6357ca299d339abc4e43ba324ad1" +
	"efe6eb1a5a6e137daa6ec9f6be30931ca368a944cfcf2a0a29f9a9664188f0466e6f078c347f9fe26a9a89d2029462b19245f24ace47aecace6ef85a4e96b31b" +
	"5f470eb0165c6375eb8f245d50a25d521d1e569e3b2dccce626752bb26eae624a24511e831a81fab6898a791579f462574ca4851e6588116493dbccc3072e0c5")
*/

func TestBtmhash() {
	fmt.Println("\n----------test btmhash--------------------")
	start := time.Now()

	//seed := seedHash(0)
	var blockNum uint64 = 1
	blockHash := make([]byte, 32)
	seed := createSeed(blockHash)
	endSeedHash := time.Now()
	deltaSeedHash := endSeedHash.Sub(start)

	cache := make([]uint32, cacheLength/4)
	generateCache(cache, seed)
	endGenerateCache := time.Now()
	deltaGenerateCache := endGenerateCache.Sub(endSeedHash)

	matList := make([]matrix.Matrix, matNum)
	fillMatrixList(matList, cache, blockNum)
	endFillMatrixList := time.Now()
	deltaFillMatrixList := endFillMatrixList.Sub(endGenerateCache)

	blockHeader := make([]byte, 80)
	matIndex := getIndex(blockHeader)

	result := mulMatrix(matList, matIndex)
	endMulMatrix := time.Now()
	deltaMulMatrix := endMulMatrix.Sub(endFillMatrixList)

	end := time.Now()
	delta := end.Sub(start)

	fmt.Println("\n---------------------------------------")
	fmt.Println("length of location is:", len(location))
	fmt.Println("result[1,1] is:", result.Get(1, 1))
	fmt.Println("cache item is:", cache[cacheLength/4-1])
	//result.Print()

	fmt.Println("\n---------------------------------------")
	fmt.Println("matSize is:", matSize)
	fmt.Println("matNum is:", matNum)
	fmt.Println("seedhash took amount of time is:", deltaSeedHash)
	fmt.Println("generateCache took amount of time is:", deltaGenerateCache)
	fmt.Println("fillMatrixList took amount of time is:", deltaFillMatrixList)
	fmt.Println("mulMatrix took amount of time is:", deltaMulMatrix)
	fmt.Println("btmhash took amount of time is:", delta)
}

func TestSHA3() {
	word := "testing"
	sum256 := sha3.Sum256([]byte(word))

	// decimal byte slice translate to hex string
	sum256Channel := make(chan []byte, 1)
	sum256Channel <- sum256[:]
	sum256_hex := DecimalByteSlice2HexString(<-sum256Channel)

	fmt.Println("\n----------test SHA3--------------------")
	fmt.Println("word is:", word)
	fmt.Println("sum256 length is:", len(sum256))
	fmt.Println("SHA3-256 sum is:", sum256)
	fmt.Println("SHA3-256 sum(hex) is:", sum256_hex)

	// testNonce := uint64(3)
	data_1 := make([]byte, 32)
	data_2 := make([]byte, 32)
	fmt.Println("data is:", data_1)

	sha3_sum256 := makeHasher(sha3.New256())
	//keccak256 := makeHasher(sha3.NewKeccak256())
	sha3_sum256(data_1, data_1)
	//keccak256(data_2, data_2)

	fmt.Println("gosha3 data is:", data_1)
	fmt.Println("sha3 is:", sha3.Sum256(data_1))
	fmt.Println("gosha3 data(hex) is:", DecimalByteSlice2HexString(data_1))
	fmt.Println("keccak256 data is:", data_2)
	fmt.Println("keccak256 data(256) is:", DecimalByteSlice2HexString(data_2))
	fmt.Println("data_1[0] is:", data_1[0])

	count := make([]int, 128)
	for i := 0; i < 12800; i++ {
		sha3_sum256(data_1, data_1)
		for j := 0; j < 32; j++ {
			count[data_1[j]%128]++
		}
	}
	fmt.Println("count is:", count)
	// fmt.Println("testNonce(uint64) is:", testNonce)

	// ----------test SHA3--------------------
	// word is: testing
	// sum256 length is: 32
	// SHA3-256 sum is: [127 89 121 251 120 240 130 232 177 198 118 99 93 184 121 92 74 198 250 186 3 82 95 183 8 203 95 214 143 212 12 94]
	// SHA3-256 sum(hex) is: 7F5979FB78F082E8B1C676635DB8795C4AC6FABA03525FB708CB5FD68FD40C5E
}

func TestSeedHash() {
	seed_1 := seedHash(0)
	seed_2 := seedHash(30000)

	//decimal byte slice translate to hex string
	seedHex_1 := DecimalByteSlice2HexString(seed_1)
	seedHex_2 := DecimalByteSlice2HexString(seed_2)

	fmt.Println("\n----------test seedhash----------------")
	fmt.Println("seed length is:", len(seed_1))
	fmt.Println("block is:", 0, "\nseedhash is:", seedHash(0))
	fmt.Println("seed(hex) is:", seedHex_1)
	fmt.Println("block is:", 30000, "\nseedhash is:", seedHash(30000))
	fmt.Println("seed(hex) is:", seedHex_2)

	// ----------test seedhash----------------
	// seed length is: 32
	// block is: 0
	// seedhash is: [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
	// seed(hex) is: 0000000000000000000000000000000000000000000000000000000000000000
	// block is: 30000
	// seedhash is: [41 13 236 217 84 139 98 168 214 3 69 169 136 56 111 200 75 166 188 149 72 64 8 246 54 47 147 22 14 243 229 99]
	// seed(hex) is: 290DECD9548B62A8D60345A988386FC84BA6BC95484008F6362F93160EF3E563
}

func TestGenerateCache() {
	start := time.Now()

	cache := make([]uint32, cacheLength/4)
	generateCache(cache, seedHash(uint64(30000)))

	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&cache))
	header.Len *= 4
	header.Cap *= 4
	cacheInt8 := *(*[]int8)(unsafe.Pointer(&header))

	end := time.Now()
	delta := end.Sub(start)

	fmt.Println("\n----------TestCacheGeneration----------")
	fmt.Println("cache[0] is:", cache[0])
	fmt.Println("cache[1] is:", cache[1])
	fmt.Println("cache length is:", len(cache))
	fmt.Println("cacheInt8[0]", cacheInt8[0])
	fmt.Println("cacheInt8[1]", cacheInt8[1])
	fmt.Println("cacheInt8[2]", cacheInt8[2])
	fmt.Println("cacheInt8[3]", cacheInt8[3])
	fmt.Println("cacheInt8[4]", cacheInt8[4])
	fmt.Println("cacheInt8[5]", cacheInt8[5])
	fmt.Println("test GenertateCache took amount of time is:", delta)
}

func TestRandomness() {
	var blockNum uint64 = 0
	seed := seedHash(30000)
	cache := make([]uint32, cacheLength/4)
	generateCache(cache, seed)
	matList := make([]matrix.Matrix, matNum)
	fillMatrixList(matList, cache, blockNum)
	matIndex := make([]byte, 80)
	ma := mulMatrix(matList, matIndex)
	nonce[0]++
	mb := mulMatrix(matList, matIndex)
	count := count_diff_bits(ma, mb)

	fmt.Println("\n----------test randomness--------------------")
	fmt.Println("ma is:")
	//ma.Print()
	fmt.Println("\n-----------------")
	fmt.Println("mb is:")
	//mb.Print()

	fmt.Println("\n------------------------------------------")
	fmt.Println("matSize is:", matSize)
	fmt.Println("matNum is:", matNum)
	fmt.Println("rate of different bits is:", float32(count)/float32(matSize*matSize*8))
}

func count_diff_bits(ma, mb *matrix.Matrix) int {
	var count int
	var mask int16
	for i := 1; i <= matSize; i++ {
		for j := 1; j <= matSize; j++ {
			for mask = 0x01; mask < 0x100; mask <<= 1 {
				if (int16(ma.Get(i, j)) & mask) != (int16(mb.Get(i, j)) & mask) {
					count++
				}
			}
		}
	}

	return count
}

func DecimalByteSlice2HexString(DecimalSlice []byte) string {
	var sa = make([]string, 0)
	for _, v := range DecimalSlice {
		sa = append(sa, fmt.Sprintf("%02X", v))
	}
	ss := strings.Join(sa, "")
	return ss
}

func Testnonce() {
	nonce := make([]uint64, 4)
	nonce[0] = 31122932229983
	nonce[1] = 31122932212982
	nonce[2] = 31132932042981
	nonce[3] = 31122932089980

	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&nonce))
	header.Len *= 8
	header.Cap *= 8
	dest := *(*[]byte)(unsafe.Pointer(&header))

	var item int8 = -3
	var matnum uint8 = 128
	var index uint8
	//var result int

	// Swap the byte order on big endian systems and return
	// if !isLittleEndian() {
	// 	swap(dest)
	// }

	fmt.Println("\n-------------test nonce-------------------------")
	fmt.Println("item is:", byte(item))
	fmt.Println("nonce is:", nonce)
	fmt.Println("matnum is:", matnum)
	fmt.Println("length of dest is:", len(dest))
	fmt.Println("dest is:", dest)
	dest = append(dest, byte(item))
	fmt.Println("length of dest is:", len(dest))
	fmt.Println("dest is:", dest)
	sum256 := sha3.Sum256(dest)
	fmt.Println("sum256 is:", sum256)
	fmt.Println("sum256[0] is:", sum256[0])
	index = uint8(sum256[0] & 0x7f)
	fmt.Println("index is:", index)

	//getIndex(item, nonce)
}
