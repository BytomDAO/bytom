package compiler

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bytom/exp/ivy/compiler/ivytest"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name     string
		contract string
		wantJSON string
	}{
		{
			"TrivialLock",
			ivytest.TrivialLock,
			`[{"name":"TrivialLock","clauses":[{"name":"trivialUnlock","values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"51","body_opcodes":"TRUE","recursive":false}]`,
		},
		{
			"LockWithPublicKey",
			ivytest.LockWithPublicKey,
			`[{"name":"LockWithPublicKey","params":[{"name":"publicKey","declared_type":"PublicKey"}],"clauses":[{"name":"unlockWithSig","params":[{"name":"sig","declared_type":"Signature"}],"values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"ae7cac","body_opcodes":"TXSIGHASH SWAP CHECKSIG","recursive":false}]`,
		},
		{
			"LockWithPublicKeyHash",
			ivytest.LockWithPKHash,
			`[{"name":"LockWithPublicKeyHash","params":[{"name":"pubKeyHash","declared_type":"Hash","inferred_type":"Sha3(PublicKey)"}],"clauses":[{"name":"spend","params":[{"name":"pubKey","declared_type":"PublicKey"},{"name":"sig","declared_type":"Signature"}],"hash_calls":[{"hash_type":"sha3","arg":"pubKey","arg_type":"PublicKey"}],"values":[{"name":"value"}]}],"value":"value","body_bytecode":"5279aa887cae7cac","body_opcodes":"2 PICK SHA3 EQUALVERIFY SWAP TXSIGHASH SWAP CHECKSIG","recursive":false}]`,
		},
		{
			"LockWith2of3Keys",
			ivytest.LockWith2of3Keys,
			`[{"name":"LockWith3Keys","params":[{"name":"pubkey1","declared_type":"PublicKey"},{"name":"pubkey2","declared_type":"PublicKey"},{"name":"pubkey3","declared_type":"PublicKey"}],"clauses":[{"name":"unlockWith2Sigs","params":[{"name":"sig1","declared_type":"Signature"},{"name":"sig2","declared_type":"Signature"}],"values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"537a547a526bae71557a536c7cad","body_opcodes":"3 ROLL 4 ROLL 2 TOALTSTACK TXSIGHASH 2ROT 5 ROLL 3 FROMALTSTACK SWAP CHECKMULTISIG","recursive":false}]`,
		},
		{
			"LockToOutput",
			ivytest.LockToOutput,
			`[{"name":"LockToOutput","params":[{"name":"address","declared_type":"Program"}],"clauses":[{"name":"relock","values":[{"name":"locked","program":"address"}]}],"value":"locked","body_bytecode":"0000c3c251557ac1","body_opcodes":"0 0 AMOUNT ASSET 1 5 ROLL CHECKOUTPUT","recursive":false}]`,
		},
		{
			"TradeOffer",
			ivytest.TradeOffer,
			`[{"name":"TradeOffer","params":[{"name":"requestedAsset","declared_type":"Asset"},{"name":"requestedAmount","declared_type":"Amount"},{"name":"sellerProgram","declared_type":"Program"},{"name":"sellerKey","declared_type":"PublicKey"}],"clauses":[{"name":"trade","reqs":[{"name":"payment","asset":"requestedAsset","amount":"requestedAmount"}],"values":[{"name":"payment","program":"sellerProgram","asset":"requestedAsset","amount":"requestedAmount"},{"name":"offered"}]},{"name":"cancel","params":[{"name":"sellerSig","declared_type":"Signature"}],"values":[{"name":"offered","program":"sellerProgram"}]}],"value":"offered","body_bytecode":"547a641300000000007251557ac16323000000547a547aae7cac690000c3c251577ac1","body_opcodes":"4 ROLL JUMPIF:$cancel $trade 0 0 2SWAP 1 5 ROLL CHECKOUTPUT JUMP:$_end $cancel 4 ROLL 4 ROLL TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 7 ROLL CHECKOUTPUT $_end","recursive":false}]`,
		},
		{
			"EscrowedTransfer",
			ivytest.EscrowedTransfer,
			`[{"name":"EscrowedTransfer","params":[{"name":"agent","declared_type":"PublicKey"},{"name":"sender","declared_type":"Program"},{"name":"recipient","declared_type":"Program"}],"clauses":[{"name":"approve","params":[{"name":"sig","declared_type":"Signature"}],"values":[{"name":"value","program":"recipient"}]},{"name":"reject","params":[{"name":"sig","declared_type":"Signature"}],"values":[{"name":"value","program":"sender"}]}],"value":"value","body_bytecode":"537a641b000000537a7cae7cac690000c3c251567ac1632a000000537a7cae7cac690000c3c251557ac1","body_opcodes":"3 ROLL JUMPIF:$reject $approve 3 ROLL SWAP TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 6 ROLL CHECKOUTPUT JUMP:$_end $reject 3 ROLL SWAP TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 5 ROLL CHECKOUTPUT $_end","recursive":false}]`,
		},
		{
			"RevealPreimage",
			ivytest.RevealPreimage,
			`[{"name":"RevealPreimage","params":[{"name":"hash","declared_type":"Hash","inferred_type":"Sha3(String)"}],"clauses":[{"name":"reveal","params":[{"name":"string","declared_type":"String"}],"hash_calls":[{"hash_type":"sha3","arg":"string","arg_type":"String"}],"values":[{"name":"value"}]}],"value":"value","body_bytecode":"7caa87","body_opcodes":"SWAP SHA3 EQUAL","recursive":false}]`,
		},
		{
			"PriceChanger",
			ivytest.PriceChanger,
			`[{"name":"PriceChanger","params":[{"name":"askAmount","declared_type":"Amount"},{"name":"askAsset","declared_type":"Asset"},{"name":"sellerKey","declared_type":"PublicKey"},{"name":"sellerProg","declared_type":"Program"}],"clauses":[{"name":"changePrice","params":[{"name":"newAmount","declared_type":"Amount"},{"name":"newAsset","declared_type":"Asset"},{"name":"sig","declared_type":"Signature"}],"values":[{"name":"offered","program":"PriceChanger(newAmount, newAsset, sellerKey, sellerProg)"}],"contracts":["PriceChanger"]},{"name":"redeem","reqs":[{"name":"payment","asset":"askAsset","amount":"askAmount"}],"values":[{"name":"payment","program":"sellerProg","asset":"askAsset","amount":"askAmount"},{"name":"offered"}]}],"value":"offered","body_bytecode":"557a6433000000557a5479ae7cac690000c3c251005a7a89597a89597a89597a89567a890274787e008901c07ec1633d0000000000537a547a51577ac1","body_opcodes":"5 ROLL JUMPIF:$redeem $changePrice 5 ROLL 4 PICK TXSIGHASH SWAP CHECKSIG VERIFY 0 0 AMOUNT ASSET 1 0 10 ROLL CATPUSHDATA 9 ROLL CATPUSHDATA 9 ROLL CATPUSHDATA 9 ROLL CATPUSHDATA 6 ROLL CATPUSHDATA 0x7478 CAT 0 CATPUSHDATA 192 CAT CHECKOUTPUT JUMP:$_end $redeem 0 0 3 ROLL 4 ROLL 1 7 ROLL CHECKOUTPUT $_end","recursive":true}]`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := strings.NewReader(c.contract)
			got, err := Compile(r)
			if err != nil {
				t.Fatal(err)
			}
			gotJSON, _ := json.Marshal(got)
			if string(gotJSON) != c.wantJSON {
				t.Errorf("\ngot  %s\nwant %s", string(gotJSON), c.wantJSON)
			} else {
				for _, contract := range got {
					t.Log(contract.Opcodes)
				}
			}
		})
	}
}

func mustDecodeHex(h string) []byte {
	bits, err := hex.DecodeString(h)
	if err != nil {
		panic(err)
	}
	return bits
}
