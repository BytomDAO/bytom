package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"bufio"
	"io/ioutil"
)

func parsePath(p *parser) []*Contract {
	path := parseImport(p)
	filename := absolutePath(string(path))
	fmt.Println("import path:", string(path))
	fmt.Println("absolute import path:", filename)

	inputFile, inputError := os.Open(filename)
	if inputError != nil {
		errmsg := fmt.Sprintf("Open the file [%v] error, err:[%v]\n", filename, inputError)
		panic(errmsg)
	}
	defer inputFile.Close()

	inputReader := bufio.NewReader(inputFile)
	inp, err := ioutil.ReadAll(inputReader)
	if err != nil {
		errmsg := fmt.Sprintf("reading input error:[%v]\n", filename, err)
		panic(errmsg)
	}

	contracts, err := parse(inp)
	if err != nil {
		errmsg := fmt.Sprintf("parse input error:[%v]\n", err)
		panic(errmsg)
	}

	var result []*Contract
	for _, contract := range contracts {
		result = append(result, contract)
	}

	return result
}

func parseImport(p *parser) []byte {
	consumeKeyword(p, "import")
	strliteral, newOffset := scanImportStr(p.buf, p.pos)
	if newOffset < 0 {
		p.errorf("Invalid import character format!")
	}
	p.pos = newOffset

	//After removing the quotes is the import filepath
	importPath := strliteral[1 : len(strliteral)-1]
	return importPath
}

func scanImportStr(buf []byte, offset int) (bytesLiteral, int) {
	offset = skipWsAndComments(buf, offset)
	if offset >= len(buf) || !(buf[offset] == '\'' || buf[offset] == '"') {
		return bytesLiteral{}, -1
	}

	for i := offset + 1; i < len(buf); i++ {
		if (buf[offset] == '\'' && buf[i] == '\'') || (buf[offset] == '"' && buf[i] == '"') {
			return bytesLiteral(buf[offset : i+1]), i + 1
		}
		if buf[i] == '\\' {
			i++
		}
	}

	panic(parseErr(buf, offset, "unterminated import string literal"))
}


func parseContractImport(p *parser) []*Contract {
	var result []*Contract
	for peekKeyword(p) == "import" {
		contracts := parsePath(p)
		for _, contract := range contracts {
			result = append(result, contract)
		}
	}
	return result
}

func absolutePath(path string) string {
	fpath, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	fpath = strings.Replace(fpath, "\\", "/", -1)

	if err := checkPath(fpath); err != nil {
		panic(err)
	}
	return fpath
}

//check whether the path is valid
func checkPath(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	return nil
}
