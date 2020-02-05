package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bytom/bytom/equity/compiler"
)

var (
	// generateInstPath is the directory (need to combine with GOPATH) for store generated contract instance
	generateInstPath = "/src/github.com/bytom/equity/instance/"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("command args: [command] [contract file_path]")
		os.Exit(0)
	}

	filename := os.Args[1]
	inputFile, inputError := os.Open(filename)
	if inputError != nil {
		fmt.Printf("An error occurred on opening the inputfile\n" +
			"Does the file exist?\n" +
			"Have you got acces to it?\n")
		os.Exit(0)
	}
	defer inputFile.Close()

	inputReader := bufio.NewReader(inputFile)
	contracts, err := compiler.Compile(inputReader)
	if err != nil {
		log.Fatal(err)
	}

	var packageName *string
	var midstr string
	var outstr []string

	//change the windows path into unix path
	filename = strings.Replace(filename, "\\", "/", -1)
	if strings.Contains(filename, "/") == true {
		outstr = strings.Split(filename, "/")
		midstr = outstr[len(outstr)-1]
	} else {
		midstr = filename
	}

	//check whether the filename contains point flag
	if strings.Contains(midstr, ".") == true {
		outstr = strings.Split(midstr, ".")
		packageName = &outstr[0]
	} else {
		packageName = &midstr
	}

	header := new(bytes.Buffer)
	fmt.Fprintf(header, "package instance\n\n")

	imports := map[string]bool{
		"bytes":                        true,
		"encoding/hex":                 true,
		"fmt":                          true,
		"github.com/bytom/equity/compiler":   true,
		"github.com/bytom/protocol/vm": true,
	}

	buf := new(bytes.Buffer)

	if len(contracts) == 1 {
		fmt.Fprintf(buf, "// %sBodyBytes refer to contract's body\n", contracts[0].Name)
		fmt.Fprintf(buf, "var %sBodyBytes []byte\n\n", contracts[0].Name)
	} else {
		fmt.Fprintf(buf, "var (\n")
		for _, contract := range contracts {
			fmt.Fprintf(buf, "\t%sBodyBytes []byte\n", contract.Name)
		}
		fmt.Fprintf(buf, ")\n\n")
	}

	fmt.Fprintf(buf, "func init() {\n")
	for _, contract := range contracts {
		fmt.Fprintf(buf, "\t%sBodyBytes, _ = hex.DecodeString(\"%x\")\n", contract.Name, contract.Body)
	}
	fmt.Fprintf(buf, "}\n\n")

	for _, contract := range contracts {
		fmt.Fprintf(buf, "// contract %s(%s) locks %s\n", contract.Name, paramsStr(contract.Params), contract.Value)
		fmt.Fprintf(buf, "//\n")
		maxWidth := 0
		for _, step := range contract.Steps {
			if len(step.Opcodes) > maxWidth {
				maxWidth = len(step.Opcodes)
			}
		}
		format := fmt.Sprintf("// %%-%d.%ds  %%s\n", maxWidth, maxWidth)
		for _, step := range contract.Steps {
			fmt.Fprintf(buf, format, step.Opcodes, step.Stack)
		}
		fmt.Fprintf(buf, "\n")

		fmt.Fprintf(buf, "// PayTo%s instantiates contract %s as a program with specific arguments.\n", contract.Name, contract.Name)
		goParams, newImports := asGoParams(contract.Params)
		for _, imp := range newImports {
			imports[imp] = true
		}
		fmt.Fprintf(buf, "func PayTo%s(%s) ([]byte, error) {\n", contract.Name, goParams)
		fmt.Fprintf(buf, "\t_contractParams := []*compiler.Param{\n")
		for _, param := range contract.Params {
			fmt.Fprintf(buf, "\t\t{Name: \"%s\", Type: \"%s\"},\n", param.Name, param.Type)
		}
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tvar _contractArgs []compiler.ContractArg\n")
		for _, param := range contract.Params {
			switch param.Type {
			case "Amount":
				fmt.Fprintf(buf, "\t_%s := int64(%s)\n", param.Name, param.Name)
				fmt.Fprintf(buf, "\t_contractArgs = append(_contractArgs, compiler.ContractArg{I: &_%s})\n", param.Name)
			case "Asset":
				fmt.Fprintf(buf, "\t_%s := %s.Bytes()\n", param.Name, param.Name)
				fmt.Fprintf(buf, "\t_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&_%s)})\n", param.Name)
			case "Boolean":
				fmt.Fprintf(buf, "\t_contractArgs = append(_contractArgs, compiler.ContractArg{B: &%s})\n", param.Name)
			case "Integer":
				fmt.Fprintf(buf, "\t_contractArgs = append(_contractArgs, compiler.ContractArg{I: &%s})\n", param.Name)
			case "Hash", "Program", "PublicKey", "Signature", "String":
				fmt.Fprintf(buf, "\t_contractArgs = append(_contractArgs, compiler.ContractArg{S: (*json.HexBytes)(&%s)})\n", param.Name)
			}
		}
		fmt.Fprintf(buf, "\treturn compiler.Instantiate(%sBodyBytes, _contractParams, %v, _contractArgs)\n", contract.Name, contract.Recursive)
		fmt.Fprintf(buf, "}\n\n")

		fmt.Fprintf(buf, "// ParsePayTo%s parses the arguments out of an instantiation of contract %s.\n", contract.Name, contract.Name)
		fmt.Fprintf(buf, "// If the input is not an instantiation of %s, returns an error.\n", contract.Name)
		fmt.Fprintf(buf, "func ParsePayTo%s(prog []byte) ([][]byte, error) {\n", contract.Name)
		fmt.Fprintf(buf, "\tvar result [][]byte\n")
		fmt.Fprintf(buf, "\tinsts, err := vm.ParseProgram(prog)\n")
		fmt.Fprintf(buf, "\tif err != nil {\n")
		fmt.Fprintf(buf, "\t\treturn nil, err\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tfor i := 0; i < %d; i++ {\n", len(contract.Params))
		fmt.Fprintf(buf, "\t\tif len(insts) == 0 {\n")
		fmt.Fprintf(buf, "\t\t\treturn nil, fmt.Errorf(\"program too short\")\n")
		fmt.Fprintf(buf, "\t\t}\n")
		fmt.Fprintf(buf, "\t\tif !insts[0].IsPushdata() {\n")
		fmt.Fprintf(buf, "\t\t\treturn nil, fmt.Errorf(\"too few arguments\")\n")
		fmt.Fprintf(buf, "\t\t}\n")
		fmt.Fprintf(buf, "\t\tresult = append(result, insts[0].Data)\n")
		fmt.Fprintf(buf, "\t\tinsts = insts[1:]\n")
		fmt.Fprintf(buf, "\t}\n")
		if contract.Recursive {
			// args... body DEPTH OVER 0 CHECKPREDICATE
			fmt.Fprintf(buf, "\tif len(insts) == 0 {\n")
			fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"program too short\")\n")
			fmt.Fprintf(buf, "\t}\n")
			fmt.Fprintf(buf, "\tif !insts[0].IsPushdata() {\n")
			fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"too few arguments\")\n")
			fmt.Fprintf(buf, "\t}\n")
			fmt.Fprintf(buf, "\tif !bytes.Equal(%sBodyBytes, insts[0].Data) {\n", contract.Name)
			fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"body bytes do not match %s\")\n", contract.Name)
			fmt.Fprintf(buf, "\t}\n")
			fmt.Fprintf(buf, "\tinsts = insts[1:]\n")
		} // else args ... DEPTH body 0 CHECKPREDICATE
		fmt.Fprintf(buf, "\tif len(insts) != 4 {\n")
		fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"program too short\")\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tif insts[0].Op != vm.OP_DEPTH {\n")
		fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"wrong program format\")\n")
		fmt.Fprintf(buf, "\t}\n")
		if contract.Recursive {
			fmt.Fprintf(buf, "\tif insts[1].Op != vm.OP_OVER {\n")
			fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"wrong program format\")\n")
			fmt.Fprintf(buf, "\t}\n")
		} else {
			fmt.Fprintf(buf, "\tif !insts[1].IsPushdata() {\n")
			fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"wrong program format\")\n")
			fmt.Fprintf(buf, "\t}\n")
			fmt.Fprintf(buf, "\tif !bytes.Equal(%sBodyBytes, insts[1].Data) {\n", contract.Name)
			fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"body bytes do not match %s\")\n", contract.Name)
			fmt.Fprintf(buf, "\t}\n")
		}
		fmt.Fprintf(buf, "\tif !insts[2].IsPushdata() {\n")
		fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"wrong program format\")\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tv, err := vm.AsInt64(insts[2].Data)\n")
		fmt.Fprintf(buf, "\tif err != nil {\n")
		fmt.Fprintf(buf, "\t\treturn nil, err\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tif v != 0 {\n")
		fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"wrong program format\")\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\tif insts[3].Op != vm.OP_CHECKPREDICATE {\n")
		fmt.Fprintf(buf, "\t\treturn nil, fmt.Errorf(\"wrong program format\")\n")
		fmt.Fprintf(buf, "\t}\n")
		fmt.Fprintf(buf, "\treturn result, nil\n")
		fmt.Fprintf(buf, "}\n\n")

		// TODO(bobg): RedeemFoo_Bar functions for marshaling the args to
		// the Bar clause of contract Foo.
	}

	fmt.Fprintf(header, "import (\n")
	for imp := range imports {
		fmt.Fprintf(header, "\t\"%s\"\n", imp)
	}
	fmt.Fprintf(header, ")\n\n")

	//get the Environment variables of GOPATH
	gopath := os.Getenv("GOPATH")
	path := gopath + generateInstPath

	//if the directory is not exist, create it
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			direrr := os.MkdirAll(path, os.ModePerm)
			if direrr != nil {
				log.Fatal(direrr)
			}
			fmt.Println("the path is create success")
		} else {
			log.Fatal(err)
		}
	}

	//store buf by create file
	file, _ := os.Create(path + *packageName + ".go")
	defer file.Close()
	file.Write(header.Bytes())
	file.Write(buf.Bytes())
	fmt.Printf("create file [%s] success!\n", *packageName+".go")
}

func paramsStr(params []*compiler.Param) string {
	var strs []string
	for _, p := range params {
		strs = append(strs, fmt.Sprintf("%s: %s", p.Name, p.Type))
	}
	return strings.Join(strs, ", ")
}

func asGoParams(params []*compiler.Param) (goParams string, imports []string) {
	var strs []string
	strFlag := false
	for _, p := range params {
		var typ string
		switch p.Type {
		case "Amount":
			typ = "uint64"
		case "Asset":
			typ = "bc.AssetID"
			imports = append(imports, "github.com/bytom/protocol/bc")
			strFlag = true
		case "Boolean":
			typ = "bool"
		case "Hash":
			typ = "[]byte"
			strFlag = true
		case "Integer":
			typ = "int64"
		case "Program":
			typ = "[]byte"
			strFlag = true
		case "PublicKey":
			typ = "ed25519.PublicKey"
			imports = append(imports, "github.com/bytom/crypto/ed25519")
			strFlag = true
		case "Signature":
			typ = "[]byte"
			strFlag = true
		case "String":
			typ = "[]byte"
			strFlag = true
		}
		strs = append(strs, fmt.Sprintf("%s %s", p.Name, typ))
	}

	if strFlag {
		imports = append(imports, "github.com/bytom/encoding/json")
	}
	return strings.Join(strs, ", "), imports
}
