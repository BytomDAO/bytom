package hexutil_test

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type MyType [5]byte

func (v *MyType) UnmarshalText(input []byte) error {
	return hexutil.UnmarshalFixedText("MyType", input, v[:])
}

func (v MyType) String() string {
	return hexutil.Bytes(v[:]).String()
}

func ExampleUnmarshalFixedText() {
	var v1, v2 MyType
	fmt.Println("v1 error:", json.Unmarshal([]byte(`"0x01"`), &v1))
	fmt.Println("v2 error:", json.Unmarshal([]byte(`"0x0101010101"`), &v2))
	fmt.Println("v2:", v2)
	// Output:
	// v1 error: hex string has length 2, want 10 for MyType
	// v2 error: <nil>
	// v2: 0x0101010101
}
