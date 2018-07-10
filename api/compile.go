package api

import (
	"strings"

	chainjson "github.com/bytom/encoding/json"
	"github.com/bytom/equity/compiler"
	"github.com/bytom/protocol/vm"
)

type (
	compileReq struct {
		Contract string                 `json:"contract"`
		Args     []compiler.ContractArg `json:"args"`
	}

	compileResp struct {
		Name    string             `json:"name"`
		Source  string             `json:"source"`
		Program chainjson.HexBytes `json:"program"`
		Params  []*compiler.Param  `json:"params"`
		Value   string             `json:"value"`
		Clauses []*compiler.Clause `json:"clause_info"`
		Opcodes string             `json:"opcodes"`
		Error   string             `json:"error"`
	}
)

func compileEquity(req compileReq) (compileResp, error) {
	var resp compileResp
	compiled, err := compiler.Compile(strings.NewReader(req.Contract))
	if err != nil {
		resp.Error = err.Error()
	}

	// if source contract maybe contain import statement, multiple contract objects will be generated
	// after the compilation, and the last object is what we need.
	contract := compiled[len(compiled)-1]
	resp = compileResp{
		Name:    contract.Name,
		Source:  req.Contract,
		Program: contract.Body,
		Value:   contract.Value,
		Clauses: contract.Clauses,
		Opcodes: contract.Opcodes,
	}

	if req.Args != nil {
		resp.Program, err = compiler.Instantiate(contract.Body, contract.Params, false, req.Args)
		if err != nil {
			resp.Error = err.Error()
		}

		resp.Opcodes, err = vm.Disassemble(resp.Program)
		if err != nil {
			return resp, err
		}
	}

	for _, param := range contract.Params {
		if param.InferredType != "" {
			param.Type = param.InferredType
			param.InferredType = ""
		}
		resp.Params = append(resp.Params, param)
	}

	return resp, nil
}

func (a *API) compileEquity(req compileReq) Response {
	resp, err := compileEquity(req)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(resp)
}
