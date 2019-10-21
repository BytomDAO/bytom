package txbuilder

import (
	"github.com/bytom/consensus"
	"github.com/bytom/consensus/segwit"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/vm/vmutil"
)

const (
	baseSize       = int64(176) // inputSize(112) + outputSize(64)
	baseP2WPKHSize = int64(98)
	baseP2WPKHGas  = int64(1409)
)

var (
	//ChainTxUtxoNum maximum utxo quantity in a tx
	ChainTxUtxoNum = 5
	//ChainTxMergeGas chain tx gas
	ChainTxMergeGas = uint64(6000000)
)

// EstimateTxGasInfo estimate transaction consumed gas
type EstimateTxGasInfo struct {
	TotalNeu    int64 `json:"total_neu"`
	FlexibleNeu int64 `json:"flexible_neu"`
	StorageNeu  int64 `json:"storage_neu"`
	VMNeu       int64 `json:"vm_neu"`
	ChainTxNeu  int64 `json:"chain_tx_neu"`
}

func EstimateChainTxGas(templates []Template) (*EstimateTxGasInfo, error) {
	estimated, err := EstimateTxGas(templates[len(templates)-1])
	if err != nil {
		return nil, err
	}

	if len(templates) > 1 {
		estimated.ChainTxNeu = int64(ChainTxMergeGas) * int64(len(templates)-1)
	}
	return estimated, nil
}

// EstimateTxGas estimate consumed neu for transaction
func EstimateTxGas(template Template) (*EstimateTxGasInfo, error) {
	var baseP2WSHSize, totalWitnessSize, baseP2WSHGas, totalP2WPKHGas, totalP2WSHGas, totalIssueGas int64
	for pos, input := range template.Transaction.TxData.Inputs {
		switch input.InputType() {
		case types.SpendInputType:
			controlProgram := input.ControlProgram()
			if segwit.IsP2WPKHScript(controlProgram) {
				totalWitnessSize += baseP2WPKHSize
				totalP2WPKHGas += baseP2WPKHGas
			} else if segwit.IsP2WSHScript(controlProgram) {
				baseP2WSHSize, baseP2WSHGas = estimateP2WSHGas(template.SigningInstructions[pos])
				totalWitnessSize += baseP2WSHSize
				totalP2WSHGas += baseP2WSHGas
			}

		case types.IssuanceInputType:
			issuanceProgram := input.IssuanceProgram()
			if height := vmutil.GetIssuanceProgramRestrictHeight(issuanceProgram); height > 0 {
				// the gas for issue program with checking block height
				totalIssueGas += 5
			}
			baseIssueSize, baseIssueGas := estimateIssueGas(template.SigningInstructions[pos])
			totalWitnessSize += baseIssueSize
			totalIssueGas += baseIssueGas
		}
	}

	flexibleGas := int64(0)
	if totalP2WPKHGas > 0 {
		flexibleGas += baseP2WPKHGas + (baseSize+baseP2WPKHSize)*consensus.StorageGasRate
	} else if totalP2WSHGas > 0 {
		flexibleGas += baseP2WSHGas + (baseSize+baseP2WSHSize)*consensus.StorageGasRate
	} else if totalIssueGas > 0 {
		totalIssueGas += baseP2WPKHGas
		totalWitnessSize += baseSize + baseP2WPKHSize
	}

	// the total transaction storage gas
	totalTxSizeGas := (int64(template.Transaction.TxData.SerializedSize) + totalWitnessSize) * consensus.StorageGasRate

	// the total transaction gas is composed of storage and virtual machines
	totalGas := totalTxSizeGas + totalP2WPKHGas + totalP2WSHGas + totalIssueGas + flexibleGas
	return &EstimateTxGasInfo{
		TotalNeu:    totalGas * consensus.VMGasRate,
		FlexibleNeu: flexibleGas * consensus.VMGasRate,
		StorageNeu:  totalTxSizeGas * consensus.VMGasRate,
		VMNeu:       (totalP2WPKHGas + totalP2WSHGas + totalIssueGas) * consensus.VMGasRate,
	}, nil
}

// estimateP2WSH return the witness size and the gas consumed to execute the virtual machine for P2WSH program
func estimateP2WSHGas(sigInst *SigningInstruction) (int64, int64) {
	var witnessSize, gas int64
	for _, witness := range sigInst.WitnessComponents {
		switch t := witness.(type) {
		case *SignatureWitness:
			witnessSize += 33*int64(len(t.Keys)) + 65*int64(t.Quorum)
			gas += 1131*int64(len(t.Keys)) + 72*int64(t.Quorum) + 659
			if int64(len(t.Keys)) == 1 && int64(t.Quorum) == 1 {
				gas += 27
			}
		case *RawTxSigWitness:
			witnessSize += 33*int64(len(t.Keys)) + 65*int64(t.Quorum)
			gas += 1131*int64(len(t.Keys)) + 72*int64(t.Quorum) + 659
			if int64(len(t.Keys)) == 1 && int64(t.Quorum) == 1 {
				gas += 27
			}
		}
	}
	return witnessSize, gas
}

// estimateIssueGas return the witness size and the gas consumed to execute the virtual machine for issuance program
func estimateIssueGas(sigInst *SigningInstruction) (int64, int64) {
	var witnessSize, gas int64
	for _, witness := range sigInst.WitnessComponents {
		switch t := witness.(type) {
		case *SignatureWitness:
			witnessSize += 65 * int64(t.Quorum)
			gas += 1065*int64(len(t.Keys)) + 72*int64(t.Quorum) + 316
		case *RawTxSigWitness:
			witnessSize += 65 * int64(t.Quorum)
			gas += 1065*int64(len(t.Keys)) + 72*int64(t.Quorum) + 316
		}
	}
	return witnessSize, gas
}
