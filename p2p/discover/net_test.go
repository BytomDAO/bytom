// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package discover

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/tendermint/go-crypto"

	"github.com/bytom/common"
)

func TestNetwork_Lookup(t *testing.T) {
	key := crypto.GenPrivKeyEd25519()
	network, err := newNetwork(lookupTestnet, key.PubKey().Unwrap().(crypto.PubKeyEd25519), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	lookupTestnet.net = network
	defer network.Close()

	// seed table with initial node (otherwise lookup will terminate immediately)
	seeds := []*Node{NewNode(lookupTestnet.dists[256][0], net.IP{10, 0, 2, 99}, lowPort+256, 999)}
	if err := network.SetFallbackNodes(seeds); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	results := network.Lookup(lookupTestnet.target)
	t.Logf("results:")
	for _, e := range results {
		fmt.Println(logdist(lookupTestnet.targetSha, hash(e.ID[:])), e.ID.String())
	}
	if len(results) != bucketSize {
		t.Errorf("wrong number of results: got %d, want %d", len(results), bucketSize)
	}
	if hasDuplicates(results) {
		t.Errorf("result set contains duplicate entries")
	}
	if !sortedByDistanceTo(lookupTestnet.targetSha, results) {
		t.Errorf("result set not sorted by distance to target")
	}
	// TODO: check result nodes are actually closest
}

// This is the test network for the Lookup test.
// The nodes were obtained by running testnet.mine with a random NodeID as target.
var lookupTestnet = &preminedTestnet{
	target:    MustHexID("166aea4f556532c6d34e8b740e5d314af7e9ac0ca79833bd751d6b665f12dfd3"),
	targetSha: common.Hash{0xe3, 0xc, 0xbb, 0xde, 0xa9, 0x91, 0xb1, 0x38, 0x2d, 0x15, 0xbd, 0x12, 0xf8, 0x18, 0xe6, 0xc0, 0x33, 0x92, 0x7a, 0x1d, 0x6b, 0x36, 0xf4, 0x90, 0xe4, 0x77, 0xe1, 0xcf, 0x2c, 0x86, 0x50, 0xde},
	dists: [257][]NodeID{
		247: []NodeID{
			MustHexID("e3ac06c25e8af5355f239c6019aca85b34876725251f3b6295c7d86513a754e0"),
			MustHexID("bb1cc114cc15693c697944c4055f8b6d703da38d66b80a6401235d5cbd2e3e1e"),
			MustHexID("dea88a1dfdf0f69a1cf2d411aefa9594302c1e18e223ca6ebccad2e52ac8e881"),
			MustHexID("4c05e26aa35041126d8ebbb723dfe98c287aa1854b09a27534cff1f90be8d25b"),
			MustHexID("54fa8624b3c0d925fb6f441e1ce6aebce86a8ce440f5f1501be26def3a3e61a0"),
			MustHexID("a38289d34a50ce84baf73bad66f3291409ede891136386346245055c452f2946"),
			MustHexID("bfbab5539348254970c9e712b0a630534a75d22d42a45281ba42113e49ece5b1"),
			MustHexID("1034429617a419b0233f85a536192e1c9cfe9d4368328525ee131963dfe2a9b7"),
			MustHexID("677f8f9cb7f6911eb68aa02c2efe3f41c1c5f8b85e6e35a5ac99255b5309effe"),
			MustHexID("755d93ee871719912861d442bd63bd59ef0c0b99f998f41007797bac2feb03f7"),
			MustHexID("f9ad86af912429ff01d3701cd9b4b7b261e4301d0b804da1e99aa2cffefae3a7"),
			MustHexID("b2275e96ea4c05635f99895b84691b23678eb7e8998e510df8e0b74b40aeb591"),
			MustHexID("bf467cc20784ef34fc11a707d03dc06bf6c7d86e85490e4393c30b36f6f41e00"),
		},
		248: []NodeID{
			MustHexID("105b2a6bbe2958f52479bf1d7934fd49413e34dedc603bb0bbaa243758f9562e"),
			MustHexID("9a7f062fa1b06804de63f6165d73fc76e21e158dbe2bba7c6697a1908b2726ad"),
			MustHexID("df1c12d3ce36c7d0abba253dfe18619e86aa954bde02cba2dc95aa739484574b"),
			MustHexID("51d6a1325d83cbba6a0bf03cbf2001cc14c69388326870fe38f4e60edbad147b"),
			MustHexID("96f087fcd81183cc5d2354c4ee34a5a4c0783cf935ce0da00ec232f845d1776e"),
			MustHexID("3569eff62af9e76fc34900658d9a7403e545eef4f05a3c64710c4d709f810414"),
			MustHexID("a701df55ad48d18dd2b404a7d92db3338e4b1bf4a4ef5d0f0801c44b8ed58634"),
			MustHexID("d6ec832503598de9ad7d05bf9d755e5c9537d768c447bef6f3e5b7d6a8830253"),
			MustHexID("51b65bc6dddb8097f32c197e331fc22c4a0a913a5843eb703fd77c2c29f1b0f6"),
			MustHexID("34cd289fc83da3b8064006ba80ad14a163ec95cb135681327b919290192b72b0"),
			MustHexID("2626202df34d9a0d1235d4f90c05256e04fabb88db065e4896e33f01d2edf7c8"),
			MustHexID("8275f6dd9b00b9da84bf103d2a0e6879b052074feaafec24a69002886ce5f4a6"),
			MustHexID("c8a8d07dfb25184ed7843f89c8c414fce052457d7c6f991c34600b122e239f5d"),
			MustHexID("cde584e105f1ec95e5f9c1027f09b5f97922045a05f7d0b3b0c9f77ac405f1a8"),
			MustHexID("a0d37c0f4d03b4d321b75f0ecd3ea4708c84057898445ef5947b33f8ebbdb943"),
			MustHexID("ff53836e466dd6465dbcb85ef085338a52314165cd5f931e82bbbe126719a013"),
		},
		249: []NodeID{
			MustHexID("8787595b7224df8e031e718036194168baa3c11a10c9e219e90114803b45a163"),
			MustHexID("df7ab187fe0af0320232b826e65e8c8ab6a9080c24bfa5fe7981233ab740a370"),
			MustHexID("01954486ca9ddd1301f56cd2968652f1677520a417a3f0a353f10c58670196b7"),
			MustHexID("98bdbc01bd68f29c246fb2d4d73521a69842dd0235b8358897c6ba977c0ad08a"),
			MustHexID("4a547a3912a3c73c34bd907eefb2e339b015a28a3370e07f27b3ff9bdfa3f2ad"),
			MustHexID("8fcf1ab1e54b30c1219d5d1de13f840e861789e3910ba568f9f82aae08533aab"),
			MustHexID("89c2485651c55bcf661a9ce06f7ed81c8c8564fd39554ef4279f4341d69c2652"),
			MustHexID("ae671cb245d543de48b0927a653096c5bb29d19d6772bdf5be459f229be9b2ba"),
			MustHexID("c5508aaea81a04bde956a89a137ecfd21c9070d13057b9833043827153de040d"),
			MustHexID("7bf9274a79c55d19cb14235b50cdd884b6f19ea3d03de3b82c3c5f7d0d87b9ba"),
			MustHexID("60f1b0d66f8becd88a2477ece12448c715213b09d715d7c1bc644065718f309e"),
			MustHexID("9ee23d8d6e4b48d53565056ab1d34f6a97e1b9c1029ae2a708518d2b9c20e051"),
			MustHexID("ae27821c543e2e8787cc5cae32a99f7d6d7067c3f27352a6a94829907212e249"),
			MustHexID("8338167914ed855eda157fbb8d98f5e0d9c7c2f428075f4d578ba657ea2493f3"),
			MustHexID("891a8e98efd8a33e875915611c2695699fe5a925f4c041f1a4fbf6943dcde583"),
			MustHexID("29174c8a03f246883b4f3b1e35f65c632e90863207c0cb0a272cfdf107ffecac"),
		},
		250: []NodeID{
			MustHexID("541291d1e532a584fbb7825ca5e674dea4b3a456852773730273a1d0e80a4eb8"),
			MustHexID("bf3fb0836cf07ac4f6bc397052e72e07d8f88668d3a5a9f7acca30a2a1dfc53e"),
			MustHexID("4c1f2323737bf1a53817287727c3e5b83cb58e80a62421bb79a4060940d2deeb"),
			MustHexID("842b7fa651af7be25036e9c92610ae4b9594d6166f6850bb0bd028f7b12d491b"),
			MustHexID("c7d212fa34da24d5a894fbf09d156a2fc754e051623932e1c79116788880b198"),
			MustHexID("90b012a06097476be50160341e2897b67d0189052e5d189a827494de7948dc79"),
			MustHexID("ad77a58ff9de7f9ad3fbf9224bdb53d22353265df2682d4cb53a9696bd904b4c"),
			MustHexID("2af1c1f2608a8d94089b1b0aaef25422a67b8edf1e7cf61a6095bf4a927c8a0d"),
			MustHexID("42d2820f2a7aa1ea9b2d925a77471cf27521d9fd5270a80089a60b07c2da71fd"),
			MustHexID("e3f3f75e73d560bb150cadeb2f560f9745133416fccc93bc6b775ae86589755d"),
			MustHexID("08a40d7b058737b875e52b02c89cf9e27cc953f911b27d93cc9f9db8f1e30dda"),
			MustHexID("7ef5afa4290ded0063608a47de889bbac363036921254352982290e7da8546c6"),
			MustHexID("6f357470d98025be39b408ab72cf403f4d82bdc78ebf6ea27845f2fd5f3f8bf3"),
			MustHexID("efc88ab3792fc043f24a30d94570268789b04b0cab35c7254aaae18eaf05a703"),
			MustHexID("f19cd22f29f588f9e977ac8577d12a99b670217a06bba223cd5fd74b167ae261"),
			MustHexID("5d640859e0af433dbde4f87297274276d4951db6ed59e2971f9287f2f4cb1774"),
		},
		251: []NodeID{
			MustHexID("7dd89e04f039efadec20f0e629b5e995be64a464785c141253513aee8030e854"),
			MustHexID("4b2a0965985233682dd81c9206b6f23b79270b49a92802869f579821ba445d3e"),
			MustHexID("bdd3ce95a92277d738ea0839b02979143ef9c4ba822e259d9dea04a4a2fa5d7a"),
			MustHexID("75273e0e17e2c601acfcf9a188dce51f13506524fef80e26976df2089d83f31e"),
			MustHexID("3b98c358275f849a8b41be845927a2602f5c84e4941d5149ee4c09386e146888"),
			MustHexID("5a9cf81ac1bf32c83fe99bd5d1493100de684a1bb667ccdf0d0df7ab9857b381"),
			MustHexID("24e24a5e3d13e92c8f96bcb952ec4db726ca2fdb63572e7dc1d1e4e31f7d17e0"),
			MustHexID("3669a74dff29a49d454ef894be4211dbcb797e424d0ce83971908e22d67d6685"),
			MustHexID("51ccb2b83cbce0a2f5639292e1652e38a4b7d0424716d8eb9864c8d07e641abb"),
			MustHexID("c3c6d9804b3abab42858e353a7f186b5668db699c7869686d3ac49537c962294"),
			MustHexID("614edcb0a4a6a2a930f70653012ca5d4ae2f793ab477e745889798380069f0a1"),
			MustHexID("dbec1ced7886736ad739a3ed6fbd99ee8bcdd978ad0c2274bd0a750a4a81c17a"),
			MustHexID("77ea9b27f257599c97f5e2fc4a63b734ab2a33881bcbb9faa558721fccca0a3b"),
			MustHexID("5b29c87ea9d9d2836fcc6c7cacff16c68b95a201d256fd758c1d16fc0a01188a"),
			MustHexID("d15d401fd676381cdce76eb57d3d47aae88e52f511b8e7f3a35cbf5b83f3c107"),
			MustHexID("89b5d694647bada3c83f24299ab09ac740b32705294c04d8dcf23d0fcfef0ae7"),
		},
		252: []NodeID{
			MustHexID("f62aefe5b2b7d6a1493674cf3ec598e2a8e2ea37a6a191242f01174fcde56e74"),
			MustHexID("7bb68333edff9b5208c74cfb0054ee2129c1659f25ef8125070c59db9726c76c"),
			MustHexID("ff9c41477e139040733fb5c9b2a1742de9ef4706b9ed3a4ec07fc7114f801d6d"),
			MustHexID("83b6d57bdb0d7d14d9736ef6d9e39628553e1ca204b58dab38cbf414b773ad38"),
			MustHexID("b338aef937f148337e442e633ea2165e3141621c4627dba2fc65c1df5e12b2f7"),
			MustHexID("d4fe63a219addb5d01617f3c9aebcb4620a93cc58a6b8cbad2df041adbf9ec44"),
			MustHexID("5e56a1a16c3f2267eead4dbe4f32644eb31ec092780b78880ca0878b11db8af0"),
			MustHexID("dad286fa3888d629563d2e1cf5c36a97f820535f68ccd481eaac59f729941d62"),
			MustHexID("e2dc2546ab325b2ecf8c09ad396eb08273e902ceb6c6422215cddd513062ae2d"),
			MustHexID("54d0ed265f13bd02051eb4212114313aec667e7958c4efdfd5ac372da3841f1b"),
			MustHexID("1a41e7011b4d7988c4d62cd44879dfd7a62157c2bb2372d425d98dd6777bcf86"),
			MustHexID("eb885eeb0f9b9d318e88dbc5fec9f6723d6ef7f60aea9c32b1fe20febaac1510"),
			MustHexID("2d31aa02694bead95000b0823767f8248f4658a7d58886d3d270e81f2938147b"),
			MustHexID("a21fc0edcfc42009e175b8f34532c9996146b55f5c60a436012d05c8d356fa86"),
			MustHexID("2352d4fa972fae5ccb2028ec182abd8266babc3c90bbc752921e83832bc4f5e0"),
			MustHexID("7458609038c1f430ed42c12bf7643c0eb9b48bb29359fc450b7201774c4148a8"),
		},
		253: []NodeID{
			MustHexID("5286209f9d328a83b2a4d6124104006e352d6ae0ccf40a9f8f05ea8511c2928b"),
			MustHexID("cc46b4c4169db2feed37a27f0de9150ce41544ab68740b8b0845d3b294072197"),
			MustHexID("58811d7c84a77848aadf294e7b1faaebbc5b62e1f66782dd38789aea30ce59d5"),
			MustHexID("e3c244a3748510e16d099a1018997f707892bc68fad140fdd864a82f5f351e9b"),
			MustHexID("91df13d79633b905ad5a20cba8224b2a19f8d85cd13d5aa6c2b254249bada066"),
			MustHexID("57cda15403d7e369b4cda5c4aa34eec51b5fc51966b6db1f2b8e5240d515bdae"),
			MustHexID("b1eecf3880bef91ec75af05a1fb554cab394446a1840d107f28f236e5e7d25f6"),
			MustHexID("8ef2f58178f53427a9f6b90da817d8e13adc57476a3cd9e7870c8a48e2a800e0"),
			MustHexID("25e8628cc73c58c8e335e7ff6bbf9e41cc60b0093b205cb6ee7fec07f4877bb3"),
			MustHexID("3a03bb273d9fe16dfb2b069d661edfba4a74b396eda018144ebe3f10cfac5c91"),
			MustHexID("531441365d6aa8f0eebefc559e42ecca58e5717bf7ee3c2e4cc545fa57eef428"),
			MustHexID("9002986fc575a2c03bfb8c06db26dfc04b0170d4b4c0c659971856935dba040d"),
			MustHexID("b525422630dadbd5046fd4b10a0788f2fe3771f3fd27efa01c82505287eb0af5"),
			MustHexID("0bb44482710edd9c12019484fb751bcf7095b2ae90ec03c1fea94f1b61ac7129"),
			MustHexID("10c2b0cad6ce2a75e211926bf32461a2bcbe68d41d0afe84ba152159ca52ec80"),
			MustHexID("eb1d4cf64b5590230268e2f5403f572186c8a8945806feb04e629628705d3cf8"),
		},
		254: []NodeID{
			MustHexID("52d2d2ffbf55fa22700c9eaae974a670a76311b55ddb8992c1c0c94e6863625e"),
			MustHexID("1154233ce268a44b0462df941dcc4a0cadbd262f792e7526d23e8aa0561b0f74"),
			MustHexID("dad29746f0e947d65f3f9268b67cbeb1ad6aed5f41087f2b78f420424f8ee584"),
			MustHexID("747f523bf966aaa5222fcaca6ee0acc144bfdce7838c68f71d040ce341ba2af3"),
			MustHexID("62620370a756c0ac3885f8a74e50b4cc51beaea77781b8cd82797ca5ccb04834"),
			MustHexID("37b85e6469b532551d4966269b8afe0dfeca6a2d69c8d6e86de117205bdb7cd5"),
			MustHexID("749bc810c92d9c5654cc6c96606098ec12b5c863b1cf0efe736aa7cc27eddf62"),
			MustHexID("279da91206e345703f273fec08a12ff3a0bdda846ea4e16f2eb99dab98bbdf4c"),
			MustHexID("b5cbb8c70852b6569da293f42fed1ce972a687351042da3b55e105b4e7a9ab74"),
			MustHexID("8ecbfe64d0c399dd79a9714e9d14a2aa4f7d2103a9be2509c3bea7641d97b3d7"),
			MustHexID("30e045d559d749246a479d19abe8214ade3fe91011aff1d7566d4e208bf2c844"),
			MustHexID("98e8ff6046b60438bf1f7a528e6b9f91da3e4d3654e3b5abc5b1c026b65dea14"),
			MustHexID("c37a442d01c4df7c7676344f3888778409dcb3f31316d4244561481d4a49b892"),
			MustHexID("2ae5175f359cb28d4109e23231049cebe57d386a31d599a245e259ad3a2a2c79"),
			MustHexID("7bee368d0b55f44f0295a737281832b249f4a52aeeadd1606bab7812088bf8dd"),
			MustHexID("37c33c9afb263b3639fb7cff4f39751e6f4cfa784fc1f31bffe83d89862b451f"),
		},
		255: []NodeID{
			MustHexID("788f7dcaeb4151867e23561ec57ed5126bdf1469171a20fe245d0f5cb736f656"),
			MustHexID("90b9aa162353dd86d1b6386098035ed2f0730e804c6dcd7197a13e56387d2e37"),
			MustHexID("406e5461e323d9f9da2b0a5bbe0e2bf17766080abe857cf301573729dcda9066"),
			MustHexID("0678eaab850e8318444564221c06211619e52924471c0750e9e25a6bbe5e3f00"),
			MustHexID("5f16607bd290a145f66aee53a1d248b43439b9c13a0291fb4605f55245a1b8df"),
			MustHexID("d75a0fc97e35499b3a34686cc35a9dd1a6750e64fa0e5db96f01a9d985c2a89c"),
			MustHexID("e3b5f916bf2803d8f37186af92d2244760aedfaa2c27c3dc1bf1ee2f2e48dbad"),
			MustHexID("0b5ea31afdeeb208ab4b58be90f2fe6897fe8d836cd69bc407236701a15390f6"),
			MustHexID("dadb88277bf84d0eed1c669bbcd5e042008f4c332e7010a4165a126734c3bed2"),
			MustHexID("5f974f5c9fd0363a9e4cc2915797cd6ad8be3ce8b5beafb1b3b2f3079366aa36"),
			MustHexID("495df4a478acd16462117f130aa63262b4a2829fd06f39c0ad505c19059095f3"),
			MustHexID("83fcfda2a77fb43c555db8fd743f99f9ebf0abd81a047919fe52ac3c4808d63c"),
			MustHexID("25b3d321513766bb55c68688808ab5930c31f1314c1eb2a061774e4bc18b1255"),
			MustHexID("906aaca169e41081bbb02a75d1f6d46b698ff457bb6fcabc4011c18e3ec2f277"),
			MustHexID("9b7733ce725cb05fcaf6dcd2df15d56ea4a5075352d9e4ce97000746fb57c102"),
			MustHexID("ef55aa4f38dd0bf1d176b97351f3ebcea3fdddac7d93f54c8a6b18ebcf607258"),
		},
		256: []NodeID{
			MustHexID("c00d05248333e3240915f045b94662945d9e62db7da1d2a549d0c0dc52274a67"),
			MustHexID("a0e93f7917510af0f2366104221c81c6c0b134f18e65af18f9f6bbac95890083"),
			MustHexID("4ba54f05578400edc10454f7baca695855fb8cd9c7dbb41809e8ea2938b46a1b"),
			MustHexID("e0b652d548d49199f3b444188c2fb7882e07d20a1737bfd526fe5b4f7fe0b019"),
			MustHexID("af6efb92c1b269a9ec18d3dd6f76552b621333f3e037f635249f0173b0f04af3"),
			MustHexID("5fd66cb1fe6a0307eace4977f38af512a0156b3efb1d0760e606471f5489d08d"),
			MustHexID("1f789e1dab0dcb6902241d7696a3b18f9bf77b6cb9d59a7f0cef5e6102c6a509"),
			MustHexID("426e5fb7f4f48b2b9de53d18b550cc3d06985bbf3ad4c599d4034b0725bfc043"),
			MustHexID("e3a52ef7396137812c9a73cb36a24b3e3c0539a4a555b44245be6ac2f93e0cb0"),
			MustHexID("8791ef49a28f15458f09be5570e7e1c38bdc4a8187e9e1f474b31193d8af40b6"),
			MustHexID("d1dda3a3466d545dfe89722ba2e6e5d3d5fd42f9230a57239dbacdeb78d011ca"),
			MustHexID("7751990831b375f73590432b8f2e5c39f4401cb197756b4cea8c36eef1d2a358"),
			MustHexID("42b513a7a361d4f98250b2cf875b00849cc8bbada00df9557b17172924cbf568"),
			MustHexID("922a630d9fe4f327780801710ed9f71a2eea437e31c790a849d5785d8bcd9a1e"),
			MustHexID("ff0168c087e931747159afe4ea9391fc15ef97321d2aed2ae05be6650140b770"),
			MustHexID("0d2aee5e0f30f82877e5f0fa1e9e6c07d623a4359fdfce241217997c7e3b2622"),
		},
	},
}

type preminedTestnet struct {
	target    NodeID
	targetSha common.Hash // sha3(target)
	dists     [hashBits + 1][]NodeID
	net       *Network
}

func (tn *preminedTestnet) sendFindnode(to *Node, target NodeID) {
	panic("sendFindnode called")
}

func (tn *preminedTestnet) sendFindnodeHash(to *Node, target common.Hash) {
	// current log distance is encoded in port number
	// fmt.Println("findnode query at dist", toaddr.Port)
	if to.UDP <= lowPort {
		panic("query to node at or below distance 0")
	}
	next := to.UDP - 1
	var result []rpcNode
	for i, id := range tn.dists[to.UDP-lowPort] {
		result = append(result, nodeToRPC(NewNode(id, net.ParseIP("10.0.2.99"), next, uint16(i)+1+lowPort)))
	}
	injectResponse(tn.net, to, neighborsPacket, &neighbors{Nodes: result})
}

func (tn *preminedTestnet) sendPing(to *Node, addr *net.UDPAddr, topics []Topic) []byte {
	injectResponse(tn.net, to, pongPacket, &pong{ReplyTok: []byte{1}})
	return []byte{1}
}

func (tn *preminedTestnet) send(to *Node, ptype nodeEvent, data interface{}) (hash []byte) {
	switch ptype {
	case pingPacket:
		injectResponse(tn.net, to, pongPacket, &pong{ReplyTok: []byte{1}})
	case pongPacket:
		// ignored
	case findnodeHashPacket:
		fmt.Println("findnodeHashPacket")
		// current log distance is encoded in port number
		// fmt.Println("findnode query at dist", toaddr.Port-lowPort)
		if to.UDP <= lowPort {
			panic("query to node at or below  distance 0")
		}
		next := to.UDP - 1
		var result []rpcNode
		for i, id := range tn.dists[to.UDP-lowPort] {
			result = append(result, nodeToRPC(NewNode(id, net.ParseIP("10.0.2.99"), next, uint16(i)+1+lowPort)))
		}
		injectResponse(tn.net, to, neighborsPacket, &neighbors{Nodes: result})
	default:
		panic("send(" + ptype.String() + ")")
	}
	return []byte{2}
}

func (tn *preminedTestnet) sendNeighbours(to *Node, nodes []*Node) {
	panic("sendNeighbours called")
}

func (tn *preminedTestnet) sendTopicQuery(to *Node, topic Topic) {
	panic("sendTopicQuery called")
}

func (tn *preminedTestnet) sendTopicNodes(to *Node, queryHash common.Hash, nodes []*Node) {
	panic("sendTopicNodes called")
}

func (tn *preminedTestnet) sendTopicRegister(to *Node, topics []Topic, idx int, pong []byte) {
	panic("sendTopicRegister called")
}

func (*preminedTestnet) Close() {}

func (*preminedTestnet) localAddr() *net.UDPAddr {
	return &net.UDPAddr{IP: net.ParseIP("10.0.1.1"), Port: 40000}
}

func TestMine(t *testing.T) {
	mine(MustHexID("166aea4f556532c6d34e8b740e5d314af7e9ac0ca79833bd751d6b665f12dfd3"))
}

// mine generates a testnet struct literal with nodes at
// various distances to the given target.
func mine(target NodeID) {
	targetSha := hash(target[:])
	var dists [hashBits + 1][]NodeID
	found := 0
	for found < bucketSize*10 {
		k := crypto.GenPrivKeyEd25519()
		id := k.PubKey().Unwrap().(crypto.PubKeyEd25519)
		sha := hash(id[:])
		ld := logdist(targetSha, sha)
		if len(dists[ld]) < bucketSize {
			dists[ld] = append(dists[ld], ByteID(id[:]))
			fmt.Println("found ID with ld", ld)
			found++
		}
	}
	fmt.Println("&preminedTestnet{")
	fmt.Printf("	target: %#v,\n", target)
	fmt.Printf("	targetSha: %#v,\n", targetSha)
	fmt.Printf("	dists: [%d][]NodeID{\n", len(dists))
	for ld, ns := range &dists {
		if len(ns) == 0 {
			continue
		}
		fmt.Printf("		%d: []NodeID{\n", ld)
		for _, n := range ns {
			fmt.Printf("			MustHexID(\"%x\"),\n", n[:])
		}
		fmt.Println("		},")
	}
	fmt.Println("	},")
	fmt.Println("}")
}

func injectResponse(net *Network, from *Node, ev nodeEvent, packet interface{}) {
	go net.reqReadPacket(ingressPacket{remoteID: from.ID, remoteAddr: from.addr(), ev: ev, data: packet})
}

func hasDuplicates(slice []*Node) bool {
	seen := make(map[NodeID]bool)
	for i, e := range slice {
		if e == nil {
			panic(fmt.Sprintf("nil *Node at %d", i))
		}
		if seen[e.ID] {
			return true
		}
		seen[e.ID] = true
	}
	return false
}

func sortedByDistanceTo(distbase common.Hash, slice []*Node) bool {
	var last common.Hash
	for i, e := range slice {
		if i > 0 && distcmp(distbase, e.sha, last) < 0 {
			return false
		}
		last = e.sha
	}
	return true
}
