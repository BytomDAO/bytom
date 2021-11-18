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
	}{}

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
