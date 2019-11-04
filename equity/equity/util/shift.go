package equity

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"github.com/bytom/bytom/protocol/vm"

	"github.com/bytom/bytom/equity/compiler"
)

const (
	firstClauseShift string = "00000000"
	endingClauseName string = "ending"
)

// Shift statistics contract clause's offset
func Shift(contract *compiler.Contract) (map[string]string, error) {
	clauseMap := make(map[string]string)
	if len(contract.Clauses) == 1 {
		clauseMap[contract.Clauses[0].Name] = firstClauseShift
		return clauseMap, nil
	}

	instructions, err := vm.ParseProgram(contract.Body)
	if err != nil {
		return nil, err
	}

	var jumpifData [][]byte
	for i, inst := range instructions {
		if inst.Op.String() == "JUMPIF" {
			if i > 0 && instructions[i-1].Op.String() == "NOP" {
				continue
			}
			jumpifData = append([][]byte{inst.Data}, jumpifData...)
		}
	}

	// Check the number of contract's clause
	if len(contract.Clauses) != len(jumpifData)+1 {
		return nil, errors.New("the number of contract's clause is not equal to the number of jumpif instruction")
	}

	for i, clause := range contract.Clauses {
		if i == 0 {
			clauseMap[clause.Name] = firstClauseShift
			continue
		}
		clauseMap[clause.Name] = hex.EncodeToString(jumpifData[i-1])
	}

	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.LittleEndian, uint32(len(contract.Body))); err != nil {
		return nil, err
	}
	clauseMap[endingClauseName] = hex.EncodeToString(buffer.Bytes())

	return clauseMap, nil
}
