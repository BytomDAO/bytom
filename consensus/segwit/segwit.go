package segwit

import (
	"errors"

	"github.com/bytom/bytom/consensus"
	"github.com/bytom/bytom/protocol/bc"
	"github.com/bytom/bytom/protocol/vm"
	"github.com/bytom/bytom/protocol/vm/vmutil"
)

func IsP2WScript(prog []byte) bool {
	return IsP2WPKHScript(prog) || IsP2WSHScript(prog) || IsStraightforward(prog)
}

func IsStraightforward(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 1 {
		return false
	}
	return insts[0].Op == vm.OP_TRUE || insts[0].Op == vm.OP_FAIL
}

func IsP2WPKHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 2 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}
	return insts[1].Op == vm.OP_DATA_20 && len(insts[1].Data) == consensus.PayToWitnessPubKeyHashDataSize
}

func IsP2WSHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 2 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}
	return insts[1].Op == vm.OP_DATA_32 && len(insts[1].Data) == consensus.PayToWitnessScriptHashDataSize
}

// IsP2WSCScript is used to determine whether it is a P2WSC script or not
func IsP2WSCScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}

	if len(insts) != 4 {
		return false
	}

	if insts[0].Op > vm.OP_16 {
		return false
	}

	for i := 1; i <= len(insts); i++ {
		if insts[i].Op != vm.OP_DATA_32 || len(insts[i].Data) != 32 {
			return false
		}
	}

	return true
}

func ConvertP2PKHSigProgram(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	if insts[0].Op == vm.OP_0 {
		return vmutil.P2PKHSigProgram(insts[1].Data)
	}
	return nil, errors.New("unknow P2PKH version number")
}

func ConvertP2SHProgram(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	if insts[0].Op == vm.OP_0 {
		return vmutil.P2SHProgram(insts[1].Data)
	}
	return nil, errors.New("unknow P2SHP version number")
}

// ConvertP2SCProgram convert standard P2WSC program into P2SC program
func ConvertP2SCProgram(prog []byte) ([]byte, error) {
	swapContractArgs, err := DecodeP2WSCProgram(prog)
	if err != nil {
		return nil, err
	}
	return vmutil.P2SCProgram(*swapContractArgs)
}

// DecodeP2WSCProgram parse standard P2WSC arguments to swapContractArgs
func DecodeP2WSCProgram(prog []byte) (*vmutil.SwapContractArgs, error) {
	if !IsP2WSCScript(prog) {
		return nil, errors.New("invalid P2WSC program")
	}

	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	swapContractArgs := &vmutil.SwapContractArgs{}
	requestedAsset := [32]byte{}
	copy(requestedAsset[:], insts[1].Data)
	swapContractArgs.RequestedAsset0 = bc.NewAssetID(requestedAsset)

	copy(requestedAsset[:], insts[2].Data)
	swapContractArgs.RequestedAsset1 = bc.NewAssetID(requestedAsset)

	copy(requestedAsset[:], insts[3].Data)
	swapContractArgs.RequestedAsset2 = bc.NewAssetID(requestedAsset)

	return swapContractArgs, nil
}

func GetHashFromStandardProg(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[1].Data, nil
}
