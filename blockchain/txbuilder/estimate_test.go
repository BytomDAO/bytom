package txbuilder

import (
	"encoding/json"
	"testing"
)

/* ------------------------------------------------------------------
the address of P2PKH VM gas:
program: 00147741752a6a989f2a72dedd966bc736b04e4bfe6f

gas         |	instructions
41			|	DUP
84			|	HASH160
29			|	DATA_20 7ee42c99f473e99bab3a774a61a0509d4efd9f09
-35			|	EQUALVERIFY
296			|	TXSIGHASH
1			|	SWAP
881			|	CHECKSIG

gas			|	witnessArguments
72			|	afcaec193fea08a74a23d50490a4f5202f534c0e0ffb09990b0c48877cb8a33f70823a18d93967b6ef5f85115cf93b0c45dbd0d83918134e8b3630a68f82dd07
40			|	b2835eb191c660193aa24ee472f6a6e41b8fa2da5cdfd9284c00c515b8ed7024

baseP2WPKHGas = 1409

------------------------------------------------------------------
the address of P2PKH VM gas: (3-2 multi-signature)
program: 0020ec0aff6b7cfee461ed40be35c3d27ddcfd923dafa50a94659f1ceb17cb12076a

npubkeys : 3
nsigs : 2

gas         						|	instructions
1 + 12 + 33*npubkeys				|	DUP
32 (npubkeys>1) / 59 (npubkeys=1)	|	SHA3
41									|	DATA_32 ec0aff6b7cfee461ed40be35c3d27ddcfd923dafa50a94659f1ceb17cb12076a
-38									|	EQUALVERIFY
17									|	DATA_8 ffffffffffffffff
1									|	SWAP
9									|	FALSE
256									|	CHECKPREDICATE

gas									|	childVM instructions
296									|	TXSIGHASH
41*npubkeys							|	DATA_32 8434df5ba6991bed993eed277c30ffd71059c02e4602e9b599e4894e279f2471
-									|	DATA_32 21461f273682a86f6307bfc187826d89173531132d5e21a13e8905bce223e28a
-									|	DATA_32 6dac720ab27734a4047c5090d9a5a1a838284430223e546f1717475c58358c44
10									|	2 02
10									|	3 03
1024*npubkeys						|	CHECKMULTISIG

gas									|	witnessArguments
72*nsigs							|	ae208434df5ba6991bed993eed277c30ffd71059c02e4602e9b599e4894e279f24712021461f273682a86f6307bfc187826d89173531132d5e21a13e8905bce223e28a206dac720ab27734a4047c5090d9a5a1a838284430223e546f1717475c58358c445253ad
-									|	a2b4d4ea7cdc8455c7f1755a2fa1233d9ab55ce377e361f4f26c899884d9e36eb874f80e68dd36d0faa67e7a5d6381228c53d17e2ebe47de87a37e6e1933180a
33*npubkeys							|	56780a70f7ae0a4a9c65b5fa4ab878a46b93f3814e9eeb40e83b1de407800262a4153e9d99efeaec36d3cb3f46ab1250c7de9f8f46b15093c79334b9c0a92601

npubkeys>1 :
baseP2WSHGas = 1131*npubkeys + 72*nsigs + 659

npubkeys=1 :
baseP2WSHGas = 1131*npubkeys + 72*nsigs + 659 + 27

------------------------------------------------------------------
the address of P2PKH VM gas: (1-1 multi-signature)
program: ae20ddc1e4243d5526383c3ae0f63f5f8a4331af575d43beca70bd489dac691d47105151ad

npubkeys : 1
nsigs : 1

gas         	|	limit with blockheight instructions(optional)
17         		|	DATA_8 00001200
17         		|	BLOCKHEIHGT
-21         	|	GREATERTHAN
-8         		|	VERIFY

totalOptional : 5

gas         	|	instructions
296         	|	TXSIGHASH
41*npubkeys     |	DATA_32 ddc1e4243d5526383c3ae0f63f5f8a4331af575d43beca70bd489dac691d4710
10         		|	1 01
10         		|	1 01
1024*npubkeys   |	CHECKMULTISIG

gas				|	witnessArguments
72*nsigs		|	ae208434df5ba6991bed993eed277c30ffd71059c02e4602e9b599e4894e279f24712021461f273682a86f6307bfc187826d89173531132d5e21a13e8905bce223e28a206dac720ab27734a4047c5090d9a5a1a838284430223e546f1717475c58358c445253ad

baseIssueGas = 1065*npubkeys + 72*nsigs + 316
------------------------------------------------------------------ */

func TestEstimateTxGas(t *testing.T) {
	cases := []struct {
		txTemplateStr   string
		wantTotalNeu    int64
		wantFlexibleNeu int64
	}{
		{
			txTemplateStr:   `{"raw_transaction":"070100010160015e9a4e2bbae57dd71b6a827fb50aaeb744ce3ae6f45c4aec7494ad097213220e8affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe0cea1bc5800011600144a6322008c5424251c7502c7d7d55f6389c3c358010001013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe086f29b3301160014fa61b0629e5f2da2bb8b08e7fc948dbd265234f700","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"19204fe9172cb0eeae86b39ec7a61ddc556656c8df08fd43ef6074296f32b347349722316972e382c339b79b7e1d83a565c6b3e7cf46847733a47044ae493257","derivation_path":["010100000000000000","0700000000000000"]}],"signatures":null},{"type":"data","value":"a527a92a7488c010bc42b39d6b50f0822183e51efab228af8ca8ca81ca459237"}]}],"allow_additional_actions":false}`,
			wantTotalNeu:    671800,
			wantFlexibleNeu: 336600,
		},
		{
			txTemplateStr:   `{"raw_transaction":"07010001016d016bcf24f1471d67c25a01ac84482ecdd8550229180171cae22321f87fe43d4f6a13ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8086a6d5f2020001220020713ef71e6087a58d6055ce81e8a8ea8a60ca19aef77923859e53a1fa9df0042989010240844b99bab9f393e89ca3bb272b1ba146852124f13a2d37fc47da6a7320f5ae1a4b6df1322750906ad480796db663e35ef7fd9544718eea08e51c5388f9813d0446ae20bd609e953918ab2ce120c43486894ff38dc4b65c2c1b4e19f6b41265d76b062120508684f922c1e5eea3dcbd592b00d297b2ddf92d35d5acabea9ff491ef514abe5152ad02014affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80bef6b4cd0201220020dc794f041d19c67108a05d2a6d797a2b12029f31b2c91ec699c9477727f25315000149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80fef9b123012200200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac6600","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"5ff7f79f0fd4eb9ccb17191b0a1ac9bed5b4a03320a06d2ff8170dd51f9ad9089c4038ec7280b5eb6745ef3d36284e67f5cf2ed2a0177d462d24abf53c0399ed","derivationPath":["010200000000000000","0400000000000000"]},{"xpub":"d0e7607bec7f68ea9135fbb9e3e94ef05a034d28be847070740fcba9454a749f6e21942cfef90f1437184cb70775beb290c13852c1497631dbcb137f74788e4f","derivationPath":["010200000000000000","0400000000000000"]}],"signatures":["","844b99bab9f393e89ca3bb272b1ba146852124f13a2d37fc47da6a7320f5ae1a4b6df1322750906ad480796db663e35ef7fd9544718eea08e51c5388f9813d04"]},{"type":"data","value":"ae20bd609e953918ab2ce120c43486894ff38dc4b65c2c1b4e19f6b41265d76b062120508684f922c1e5eea3dcbd592b00d297b2ddf92d35d5acabea9ff491ef514abe5152ad"}]}],"allow_additional_actions":false}`,
			wantTotalNeu:    1366400,
			wantFlexibleNeu: 660000,
		},
		{
			txTemplateStr:   `{"raw_transaction":"07010002016c016acf24f1471d67c25a01ac84482ecdd8550229180171cae22321f87fe43d4f6a13ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80b4c4c32101012200200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66ef020440f5baa1530bd7ded5c37f1c91360e28e736c91a7933eff961d68eebf90bdce63eb4361689759a8aa420256af565e38921985026de8d27dd7b66f0d01c90170a0440b23b44f62f3e97bcbd5f80cb9bb3d63cb154c62d402851e5b4d5d89849fef74271c8c38f594b944b75222d06ef18bddec4b6278ad3185f72ac5321ce5948e90940a00b096eef5b3bed5c6a2843d29e1820ef1413947d3e278c21cc70976c47976d1159468f071bf853b244be8f6cc55d78615ea6594c946f1a6e6622d8e9d42206a901ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad016c016a158f56c5673a52876bbbed4cd8724428b43a8d9ddd2a759c9df06b46898f101affffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80fef9b12301012200200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66ef020440c3c4fdbe99f9266a42df767cf03c22d9d09096446a8882b9d0c0076d9c85da28add31320705452fb566a091515cedb1ea9966647201236a0da13a020f848b8084043e22fe631cee95e3185ecd0c6fc4a262689d674725abe7d7f3158d8d43c776338edeec76600776fc0dcee280bd7a1a8a2b23909c6cefa7fbb55c27522b6100640fefe403941035a66ba9b6d097dfe0ada68ae6d006272928fad2ba23341fe878690e9e2fa1d2d3992c16aa20125fb2da7f7687920c12a36e4964533ceeccd3602a901ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad020149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80ea8ed51f01220020036f3d1665dc802fd36aded656c2f4b2b2c5b00e86c44f5352257b718941a4e9000149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80fef9b12301220020e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee00","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":3,"keys":[{"xpub":"5ff7f79f0fd4eb9ccb17191b0a1ac9bed5b4a03320a06d2ff8170dd51f9ad9089c4038ec7280b5eb6745ef3d36284e67f5cf2ed2a0177d462d24abf53c0399ed","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"7d1c7a9094ab23f432e60afbbfe2791ba9ab3daba8aaa544634218243b8659985cb0ae9fe2b0f5da8a84c6b117c9491bf38f5e59b0d05642d90ba34cf7611eec","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"b0d2d90cdee01976d51b55963ae214493708d8db44f7516d2d4853a542cba4c07fbd0ad3e7a9ff4b6fbe6b71e66f4538a9424eaf15f538d958aa7025f5f752dc","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"d0e7607bec7f68ea9135fbb9e3e94ef05a034d28be847070740fcba9454a749f6e21942cfef90f1437184cb70775beb290c13852c1497631dbcb137f74788e4f","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"e18b9d219e960d761e8d03290acddb5211fea1140c87663908ea74f212763ca8d809bb0fe861884e662429564fa0f2725b5787175054c17685a83a68e160344d","derivationPath":["010300000000000000","0100000000000000"]}],"signatures":["","f5baa1530bd7ded5c37f1c91360e28e736c91a7933eff961d68eebf90bdce63eb4361689759a8aa420256af565e38921985026de8d27dd7b66f0d01c90170a04","b23b44f62f3e97bcbd5f80cb9bb3d63cb154c62d402851e5b4d5d89849fef74271c8c38f594b944b75222d06ef18bddec4b6278ad3185f72ac5321ce5948e909","a00b096eef5b3bed5c6a2843d29e1820ef1413947d3e278c21cc70976c47976d1159468f071bf853b244be8f6cc55d78615ea6594c946f1a6e6622d8e9d42206",""]},{"type":"data","value":"ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad"}]},{"position":1,"witness_components":[{"type":"raw_tx_signature","quorum":3,"keys":[{"xpub":"5ff7f79f0fd4eb9ccb17191b0a1ac9bed5b4a03320a06d2ff8170dd51f9ad9089c4038ec7280b5eb6745ef3d36284e67f5cf2ed2a0177d462d24abf53c0399ed","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"7d1c7a9094ab23f432e60afbbfe2791ba9ab3daba8aaa544634218243b8659985cb0ae9fe2b0f5da8a84c6b117c9491bf38f5e59b0d05642d90ba34cf7611eec","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"b0d2d90cdee01976d51b55963ae214493708d8db44f7516d2d4853a542cba4c07fbd0ad3e7a9ff4b6fbe6b71e66f4538a9424eaf15f538d958aa7025f5f752dc","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"d0e7607bec7f68ea9135fbb9e3e94ef05a034d28be847070740fcba9454a749f6e21942cfef90f1437184cb70775beb290c13852c1497631dbcb137f74788e4f","derivationPath":["010300000000000000","0100000000000000"]},{"xpub":"e18b9d219e960d761e8d03290acddb5211fea1140c87663908ea74f212763ca8d809bb0fe861884e662429564fa0f2725b5787175054c17685a83a68e160344d","derivationPath":["010300000000000000","0100000000000000"]}],"signatures":["","c3c4fdbe99f9266a42df767cf03c22d9d09096446a8882b9d0c0076d9c85da28add31320705452fb566a091515cedb1ea9966647201236a0da13a020f848b808","43e22fe631cee95e3185ecd0c6fc4a262689d674725abe7d7f3158d8d43c776338edeec76600776fc0dcee280bd7a1a8a2b23909c6cefa7fbb55c27522b61006","fefe403941035a66ba9b6d097dfe0ada68ae6d006272928fad2ba23341fe878690e9e2fa1d2d3992c16aa20125fb2da7f7687920c12a36e4964533ceeccd3602",""]},{"type":"data","value":"ae20d441b6f375659325a04eede4fc3b74579bb08ccd05b41b99776501e22d6dca7320af6d98ca2c3cd10bf0affbfa6e86609b750523cfadb662ec963c164f05798a49209820b9f1553b03aaebe7e3f9e9222ed7db73b5079b18564042fd3b2cef74156a20271b52de5f554aa4a6f1358f1c2193617bfb3fed4546d13c4af773096a429f9420eeb4a78d8b5cb8283c221ca2d3fd96b8946b3cddee02b7ceffb8f605932588595355ad"}]}],"allow_additional_actions":false}`,
			wantTotalNeu:    4392200,
			wantFlexibleNeu: 1413200,
		},
		{
			txTemplateStr:   `{"raw_transaction":"0701dfd5c8d505020160015eb0fdbdb00567080bf5732fe4c5027478d8f013f89fc852e3ae3d7f56f5657f71ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc03020116001416845959fd7bc6edd959f9a5f7cbfcf56630cfdf01000160015ec757e7a85beafaf620112b2dd01980609f3378e9e77b5f699c969d146c307948ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc03030116001416845959fd7bc6edd959f9a5f7cbfcf56630cfdf010001013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80a8d6b907011600147741752a6a989f2a72dedd966bc736b04e4bfe6f00","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"69c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a"}]},{"position":1,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"69c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a"}]}],"fee":0,"allow_additional_actions":false}`,
			wantTotalNeu:    994000,
			wantFlexibleNeu: 336600,
		},
		{
			txTemplateStr:   `{"raw_transaction":"0701dfd5c8d505030160015ec757e7a85beafaf620112b2dd01980609f3378e9e77b5f699c969d146c307948ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc03010116001416845959fd7bc6edd959f9a5f7cbfcf56630cfdf01000160015eb0fdbdb00567080bf5732fe4c5027478d8f013f89fc852e3ae3d7f56f5657f71ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc03020116001416845959fd7bc6edd959f9a5f7cbfcf56630cfdf01000160015ec757e7a85beafaf620112b2dd01980609f3378e9e77b5f699c969d146c307948ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc03030116001416845959fd7bc6edd959f9a5f7cbfcf56630cfdf010001013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80bcc1960b011600147741752a6a989f2a72dedd966bc736b04e4bfe6f00","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"69c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a"}]},{"position":1,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"69c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a"}]},{"position":2,"witness_components":[{"type":"raw_tx_signature","quorum":1,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"69c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a"}]}],"fee":0,"allow_additional_actions":false}`,
			wantTotalNeu:    1315400,
			wantFlexibleNeu: 336600,
		},
		{
			txTemplateStr:   `{"raw_transaction":"0701dfd5c8d50502016c016a2b58638987a57138bf3022c37e1b0f7177b8593416eefbed853d4ad5301193ebffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030101220020ec0aff6b7cfee461ed40be35c3d27ddcfd923dafa50a94659f1ceb17cb12076a0100016c016a988c49f1234b5e117d0238e52f8b9def88c2b16d66bc0e4ce7a2ca3cbd3f316cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030101220020ec0aff6b7cfee461ed40be35c3d27ddcfd923dafa50a94659f1ceb17cb12076a010001013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80a8d6b907011600147741752a6a989f2a72dedd966bc736b04e4bfe6f00","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":2,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"0e95125e7e1f49887e49c05e0a505d0a8849d6ef124bf4d7cc5e2a0272b49b9674b8304f8b4d63182526beebba5009c72a712d7d5573ae0d7361a3218dba16a7","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"dab73c5f6367e87e6c537229d03172041cbd3795682df49e03a82a148655ad05926ee2236be57ac673343cd0565fe75e140d74577036d02960925801f661e3b3","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"ae2069c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a20cfedc0f57c1ab57355b2374d14bf7bbf59376c73880798cc7f2301bda2858bc620a381dffb3be8e7449b09f190a5eb39415fa46c93c99c07600572e66fbb872ee35253ad"}]},{"position":1,"witness_components":[{"type":"raw_tx_signature","quorum":2,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"0e95125e7e1f49887e49c05e0a505d0a8849d6ef124bf4d7cc5e2a0272b49b9674b8304f8b4d63182526beebba5009c72a712d7d5573ae0d7361a3218dba16a7","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"dab73c5f6367e87e6c537229d03172041cbd3795682df49e03a82a148655ad05926ee2236be57ac673343cd0565fe75e140d74577036d02960925801f661e3b3","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"ae2069c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a20cfedc0f57c1ab57355b2374d14bf7bbf59376c73880798cc7f2301bda2858bc620a381dffb3be8e7449b09f190a5eb39415fa46c93c99c07600572e66fbb872ee35253ad"}]}],"fee":0,"allow_additional_actions":false}`,
			wantTotalNeu:    2749600,
			wantFlexibleNeu: 920200,
		},
		{
			txTemplateStr:   `{"raw_transaction":"0701dfd5c8d50503016c016a988c49f1234b5e117d0238e52f8b9def88c2b16d66bc0e4ce7a2ca3cbd3f316cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030301220020ec0aff6b7cfee461ed40be35c3d27ddcfd923dafa50a94659f1ceb17cb12076a0100016c016a2b58638987a57138bf3022c37e1b0f7177b8593416eefbed853d4ad5301193ebffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030101220020ec0aff6b7cfee461ed40be35c3d27ddcfd923dafa50a94659f1ceb17cb12076a0100016c016a988c49f1234b5e117d0238e52f8b9def88c2b16d66bc0e4ce7a2ca3cbd3f316cffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030101220020ec0aff6b7cfee461ed40be35c3d27ddcfd923dafa50a94659f1ceb17cb12076a010001013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80bcc1960b011600147741752a6a989f2a72dedd966bc736b04e4bfe6f00","signing_instructions":[{"position":0,"witness_components":[{"type":"raw_tx_signature","quorum":2,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"0e95125e7e1f49887e49c05e0a505d0a8849d6ef124bf4d7cc5e2a0272b49b9674b8304f8b4d63182526beebba5009c72a712d7d5573ae0d7361a3218dba16a7","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"dab73c5f6367e87e6c537229d03172041cbd3795682df49e03a82a148655ad05926ee2236be57ac673343cd0565fe75e140d74577036d02960925801f661e3b3","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"ae2069c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a20cfedc0f57c1ab57355b2374d14bf7bbf59376c73880798cc7f2301bda2858bc620a381dffb3be8e7449b09f190a5eb39415fa46c93c99c07600572e66fbb872ee35253ad"}]},{"position":1,"witness_components":[{"type":"raw_tx_signature","quorum":2,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"0e95125e7e1f49887e49c05e0a505d0a8849d6ef124bf4d7cc5e2a0272b49b9674b8304f8b4d63182526beebba5009c72a712d7d5573ae0d7361a3218dba16a7","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"dab73c5f6367e87e6c537229d03172041cbd3795682df49e03a82a148655ad05926ee2236be57ac673343cd0565fe75e140d74577036d02960925801f661e3b3","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"ae2069c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a20cfedc0f57c1ab57355b2374d14bf7bbf59376c73880798cc7f2301bda2858bc620a381dffb3be8e7449b09f190a5eb39415fa46c93c99c07600572e66fbb872ee35253ad"}]},{"position":2,"witness_components":[{"type":"raw_tx_signature","quorum":2,"keys":[{"xpub":"f6ce12127df9f062ac3fb91836cd0ac0b7ed9f384df45e1900ed8bde6e37d98c246afd0ffa2e23a349ed5da6cc49ca2866ba38a40d51d51b4ce526327456953b","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"0e95125e7e1f49887e49c05e0a505d0a8849d6ef124bf4d7cc5e2a0272b49b9674b8304f8b4d63182526beebba5009c72a712d7d5573ae0d7361a3218dba16a7","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]},{"xpub":"dab73c5f6367e87e6c537229d03172041cbd3795682df49e03a82a148655ad05926ee2236be57ac673343cd0565fe75e140d74577036d02960925801f661e3b3","derivation_path":["2c000000","99000000","02000000","00000000","01000000"]}],"signatures":null},{"type":"data","value":"ae2069c0a1008de2dfc39fc6630c8ab4b47e0184ac7e64fd5ea4fab38f60cecc921a20cfedc0f57c1ab57355b2374d14bf7bbf59376c73880798cc7f2301bda2858bc620a381dffb3be8e7449b09f190a5eb39415fa46c93c99c07600572e66fbb872ee35253ad"}]}],"fee":0,"allow_additional_actions":false}`,
			wantTotalNeu:    3657000,
			wantFlexibleNeu: 920200,
		},
	}

	for _, c := range cases {
		template := Template{}
		err := json.Unmarshal([]byte(c.txTemplateStr), &template)
		if err != nil {
			t.Fatal(err)
		}

		estimateTxGasResp, err := EstimateTxGas(template)
		if estimateTxGasResp.TotalNeu != c.wantTotalNeu {
			t.Errorf(`got TotalNeu =%#v; want=%#v`, estimateTxGasResp.TotalNeu, c.wantTotalNeu)
		}

		if estimateTxGasResp.FlexibleNeu != c.wantFlexibleNeu {
			t.Errorf(`got FlexibleNeu =%#v; want=%#v`, estimateTxGasResp.FlexibleNeu, c.wantFlexibleNeu)
		}
	}
}
