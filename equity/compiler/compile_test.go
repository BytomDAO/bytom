package compiler

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bytom/equity/compiler/equitytest"
)

func TestCompile(t *testing.T) {
	cases := []struct {
		name     string
		contract string
		wantJSON string
	}{
		{
			"TrivialLock",
			equitytest.TrivialLock,
			`[{"name":"TrivialLock","clauses":[{"name":"trivialUnlock","values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"51","body_opcodes":"TRUE","recursive":false}]`,
		},
		{
			"LockWithPublicKey",
			equitytest.LockWithPublicKey,
			`[{"name":"LockWithPublicKey","params":[{"name":"publicKey","type":"PublicKey"}],"clauses":[{"name":"unlockWithSig","params":[{"name":"sig","type":"Signature"}],"values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"ae7cac","body_opcodes":"TXSIGHASH SWAP CHECKSIG","recursive":false}]`,
		},
		{
			"LockWithPublicKeyHash",
			equitytest.LockWithPKHash,
			`[{"name":"LockWithPublicKeyHash","params":[{"name":"pubKeyHash","type":"Hash","inferred_type":"Sha3(PublicKey)"}],"clauses":[{"name":"spend","params":[{"name":"pubKey","type":"PublicKey"},{"name":"sig","type":"Signature"}],"hash_calls":[{"hash_type":"sha3","arg":"pubKey","arg_type":"PublicKey"}],"values":[{"name":"value"}]}],"value":"value","body_bytecode":"5279aa887cae7cac","body_opcodes":"2 PICK SHA3 EQUALVERIFY SWAP TXSIGHASH SWAP CHECKSIG","recursive":false}]`,
		},
		{
			"LockWith2of3Keys",
			equitytest.LockWith2of3Keys,
			`[{"name":"LockWith3Keys","params":[{"name":"pubkey1","type":"PublicKey"},{"name":"pubkey2","type":"PublicKey"},{"name":"pubkey3","type":"PublicKey"}],"clauses":[{"name":"unlockWith2Sigs","params":[{"name":"sig1","type":"Signature"},{"name":"sig2","type":"Signature"}],"values":[{"name":"locked"}]}],"value":"locked","body_bytecode":"537a547a526bae71557a536c7cad","body_opcodes":"3 ROLL 4 ROLL 2 TOALTSTACK TXSIGHASH 2ROT 5 ROLL 3 FROMALTSTACK SWAP CHECKMULTISIG","recursive":false}]`,
		},
		{
			"LockToOutput",
			equitytest.LockToOutput,
			`[{"name":"LockToOutput","params":[{"name":"address","type":"Program"}],"clauses":[{"name":"relock","values":[{"name":"locked","program":"address"}]}],"value":"locked","body_bytecode":"00c3c251547ac1","body_opcodes":"0 AMOUNT ASSET 1 4 ROLL CHECKOUTPUT","recursive":false}]`,
		},
		{
			"TradeOffer",
			equitytest.TradeOffer,
			`[{"name":"TradeOffer","params":[{"name":"requestedAsset","type":"Asset"},{"name":"requestedAmount","type":"Amount"},{"name":"sellerProgram","type":"Program"},{"name":"sellerKey","type":"PublicKey"}],"clauses":[{"name":"trade","reqs":[{"name":"payment","asset":"requestedAsset","amount":"requestedAmount"}],"values":[{"name":"payment","program":"sellerProgram","asset":"requestedAsset","amount":"requestedAmount"},{"name":"offered"}]},{"name":"cancel","params":[{"name":"sellerSig","type":"Signature"}],"values":[{"name":"offered","program":"sellerProgram"}]}],"value":"offered","body_bytecode":"547a6413000000007b7b51547ac16322000000547a547aae7cac6900c3c251567ac1","body_opcodes":"4 ROLL JUMPIF:$cancel $trade 0 ROT ROT 1 4 ROLL CHECKOUTPUT JUMP:$_end $cancel 4 ROLL 4 ROLL TXSIGHASH SWAP CHECKSIG VERIFY 0 AMOUNT ASSET 1 6 ROLL CHECKOUTPUT $_end","recursive":false}]`,
		},
		{
			"EscrowedTransfer",
			equitytest.EscrowedTransfer,
			`[{"name":"EscrowedTransfer","params":[{"name":"agent","type":"PublicKey"},{"name":"sender","type":"Program"},{"name":"recipient","type":"Program"}],"clauses":[{"name":"approve","params":[{"name":"sig","type":"Signature"}],"values":[{"name":"value","program":"recipient"}]},{"name":"reject","params":[{"name":"sig","type":"Signature"}],"values":[{"name":"value","program":"sender"}]}],"value":"value","body_bytecode":"537a641a000000537a7cae7cac6900c3c251557ac16328000000537a7cae7cac6900c3c251547ac1","body_opcodes":"3 ROLL JUMPIF:$reject $approve 3 ROLL SWAP TXSIGHASH SWAP CHECKSIG VERIFY 0 AMOUNT ASSET 1 5 ROLL CHECKOUTPUT JUMP:$_end $reject 3 ROLL SWAP TXSIGHASH SWAP CHECKSIG VERIFY 0 AMOUNT ASSET 1 4 ROLL CHECKOUTPUT $_end","recursive":false}]`,
		},
		{
			"CollateralizedLoan",
			equitytest.CollateralizedLoan,
			`[{"name":"CollateralizedLoan","params":[{"name":"balanceAsset","type":"Asset"},{"name":"balanceAmount","type":"Amount"},{"name":"finalHeight","type":"Integer"},{"name":"lender","type":"Program"},{"name":"borrower","type":"Program"}],"clauses":[{"name":"repay","reqs":[{"name":"payment","asset":"balanceAsset","amount":"balanceAmount"}],"values":[{"name":"payment","program":"lender","asset":"balanceAsset","amount":"balanceAmount"},{"name":"collateral","program":"borrower"}]},{"name":"default","blockheight":["finalHeight"],"values":[{"name":"collateral","program":"lender"}]}],"value":"collateral","body_bytecode":"557a641b000000007b7b51557ac16951c3c251557ac163260000007bcd9f6900c3c251567ac1","body_opcodes":"5 ROLL JUMPIF:$default $repay 0 ROT ROT 1 5 ROLL CHECKOUTPUT VERIFY 1 AMOUNT ASSET 1 5 ROLL CHECKOUTPUT JUMP:$_end $default ROT BLOCKHEIGHT LESSTHAN VERIFY 0 AMOUNT ASSET 1 6 ROLL CHECKOUTPUT $_end","recursive":false}]`,
		},
		{
			"RevealPreimage",
			equitytest.RevealPreimage,
			`[{"name":"RevealPreimage","params":[{"name":"hash","type":"Hash","inferred_type":"Sha3(String)"}],"clauses":[{"name":"reveal","params":[{"name":"string","type":"String"}],"hash_calls":[{"hash_type":"sha3","arg":"string","arg_type":"String"}],"values":[{"name":"value"}]}],"value":"value","body_bytecode":"7caa87","body_opcodes":"SWAP SHA3 EQUAL","recursive":false}]`,
		},
		{
			"CallOptionWithSettlement",
			equitytest.CallOptionWithSettlement,
			`[{"name":"CallOptionWithSettlement","params":[{"name":"strikePrice","type":"Amount"},{"name":"strikeCurrency","type":"Asset"},{"name":"sellerProgram","type":"Program"},{"name":"sellerKey","type":"PublicKey"},{"name":"buyerKey","type":"PublicKey"},{"name":"finalHeight","type":"Integer"}],"clauses":[{"name":"exercise","params":[{"name":"buyerSig","type":"Signature"}],"reqs":[{"name":"payment","asset":"strikeCurrency","amount":"strikePrice"}],"blockheight":["finalHeight"],"values":[{"name":"payment","program":"sellerProgram","asset":"strikeCurrency","amount":"strikePrice"},{"name":"underlying"}]},{"name":"expire","blockheight":["finalHeight"],"values":[{"name":"underlying","program":"sellerProgram"}]},{"name":"settle","params":[{"name":"sellerSig","type":"Signature"},{"name":"buyerSig","type":"Signature"}],"values":[{"name":"underlying"}]}],"value":"underlying","body_bytecode":"567a76529c64360000006425000000557acda06971ae7cac69007c7b51547ac16346000000557acd9f6900c3c251567ac1634600000075577a547aae7cac69557a547aae7cac","body_opcodes":"6 ROLL DUP 2 NUMEQUAL JUMPIF:$settle JUMPIF:$expire $exercise 5 ROLL BLOCKHEIGHT GREATERTHAN VERIFY 2ROT TXSIGHASH SWAP CHECKSIG VERIFY 0 SWAP ROT 1 4 ROLL CHECKOUTPUT JUMP:$_end $expire 5 ROLL BLOCKHEIGHT LESSTHAN VERIFY 0 AMOUNT ASSET 1 6 ROLL CHECKOUTPUT JUMP:$_end $settle DROP 7 ROLL 4 ROLL TXSIGHASH SWAP CHECKSIG VERIFY 5 ROLL 4 ROLL TXSIGHASH SWAP CHECKSIG $_end","recursive":false}]`,
		},
		{
			"PriceChanger",
			equitytest.PriceChanger,
			`[{"name":"PriceChanger","params":[{"name":"askAmount","type":"Amount"},{"name":"askAsset","type":"Asset"},{"name":"sellerKey","type":"PublicKey"},{"name":"sellerProg","type":"Program"}],"clauses":[{"name":"changePrice","params":[{"name":"newAmount","type":"Amount"},{"name":"newAsset","type":"Asset"},{"name":"sig","type":"Signature"}],"values":[{"name":"offered","program":"PriceChanger(newAmount, newAsset, sellerKey, sellerProg)"}],"contracts":["PriceChanger"]},{"name":"redeem","reqs":[{"name":"payment","asset":"askAsset","amount":"askAmount"}],"values":[{"name":"payment","program":"sellerProg","asset":"askAsset","amount":"askAmount"},{"name":"offered"}]}],"value":"offered","body_bytecode":"557a6432000000557a5479ae7cac6900c3c25100597a89587a89587a89587a89557a890274787e008901c07ec1633a000000007b537a51567ac1","body_opcodes":"5 ROLL JUMPIF:$redeem $changePrice 5 ROLL 4 PICK TXSIGHASH SWAP CHECKSIG VERIFY 0 AMOUNT ASSET 1 0 9 ROLL CATPUSHDATA 8 ROLL CATPUSHDATA 8 ROLL CATPUSHDATA 8 ROLL CATPUSHDATA 5 ROLL CATPUSHDATA 0x7478 CAT 0 CATPUSHDATA 192 CAT CHECKOUTPUT JUMP:$_end $redeem 0 ROT 3 ROLL 1 6 ROLL CHECKOUTPUT $_end","recursive":true}]`,
		},
		{
			"OneTwo",
			equitytest.OneTwo,
			`[{"name":"Two","params":[{"name":"b","type":"Program"},{"name":"c","type":"Program"},{"name":"expirationHeight","type":"Integer"}],"clauses":[{"name":"redeem","blockheight":["expirationHeight"],"values":[{"name":"value","program":"b"}]},{"name":"default","blockheight":["expirationHeight"],"values":[{"name":"value","program":"c"}]}],"value":"value","body_bytecode":"537a64170000007bcda06900c3c251547ac163220000007bcd9f6900c3c251557ac1","body_opcodes":"3 ROLL JUMPIF:$default $redeem ROT BLOCKHEIGHT GREATERTHAN VERIFY 0 AMOUNT ASSET 1 4 ROLL CHECKOUTPUT JUMP:$_end $default ROT BLOCKHEIGHT LESSTHAN VERIFY 0 AMOUNT ASSET 1 5 ROLL CHECKOUTPUT $_end","recursive":false},{"name":"One","params":[{"name":"a","type":"Program"},{"name":"b","type":"Program"},{"name":"c","type":"Program"},{"name":"switchHeight","type":"Integer"},{"name":"blockHeight","type":"Integer"}],"clauses":[{"name":"redeem","blockheight":["switchHeight"],"values":[{"name":"value","program":"a"}]},{"name":"switch","blockheight":["switchHeight"],"values":[{"name":"value","program":"Two(b, c, blockHeight)"}],"contracts":["Two"]}],"value":"value","body_bytecode":"557a6418000000537acda06900c3c251547ac16358000000537acd9f6900c3c25100587a89577a89567a8901747e22537a64170000007bcda06900c3c251547ac163220000007bcd9f6900c3c251557ac189008901c07ec1","body_opcodes":"5 ROLL JUMPIF:$switch $redeem 3 ROLL BLOCKHEIGHT GREATERTHAN VERIFY 0 AMOUNT ASSET 1 4 ROLL CHECKOUTPUT JUMP:$_end $switch 3 ROLL BLOCKHEIGHT LESSTHAN VERIFY 0 AMOUNT ASSET 1 0 8 ROLL CATPUSHDATA 7 ROLL CATPUSHDATA 6 ROLL CATPUSHDATA 116 CAT 0x537a64170000007bcda06900c3c251547ac163220000007bcd9f6900c3c251557ac1 CATPUSHDATA 0 CATPUSHDATA 192 CAT CHECKOUTPUT $_end","recursive":false}]`,
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
