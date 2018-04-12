package test

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var (
	baseDir       = filepath.Join(".", "testdata")
	walletTestDir = filepath.Join(baseDir, "wallet_tests")
	chainTestDir  = filepath.Join(baseDir, "chain_tests")
	txTestDir     = filepath.Join(baseDir, "tx_tests")
)

func readJSON(reader io.Reader, value interface{}) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("error reading JSON file: %v", err)
	}
	if err = json.Unmarshal(data, &value); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(data, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at line %v: %v", line, err)
		}
		return err
	}
	return nil
}

func readJSONFile(fn string, value interface{}) error {
	file, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := readJSON(file, value); err != nil {
		return fmt.Errorf("%s in file %s", err.Error(), fn)
	}
	return nil
}

// findLine returns the line number for the given offset into data.
func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}

// walk invokes its runTest argument for all subtests in the given directory.
//
// runTest should be a function of type func(t *testing.T, name string, x <TestType>),
// where TestType is the type of the test contained in test files.
func walk(t *testing.T, dir string, runTest interface{}) {
	// Walk the directory.
	dirinfo, err := os.Stat(dir)
	if os.IsNotExist(err) || !dirinfo.IsDir() {
		fmt.Fprintf(os.Stderr, "can't find test files in %s\n", dir)
		t.Skip("missing test files")
	}
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		name := filepath.ToSlash(strings.TrimPrefix(path, dir+string(filepath.Separator)))
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".json" {
			t.Run(name, func(t *testing.T) { runTestFile(t, path, name, runTest) })
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runTestFile(t *testing.T, path, name string, runTest interface{}) {
	m := makeMapFromTestFunc(runTest)
	if err := readJSONFile(path, m.Addr().Interface()); err != nil {
		t.Fatal(err)
	}
	runTestFunc(runTest, t, name, m)
}

func makeMapFromTestFunc(f interface{}) reflect.Value {
	stringT := reflect.TypeOf("")
	testingT := reflect.TypeOf((*testing.T)(nil))
	ftyp := reflect.TypeOf(f)
	if ftyp.Kind() != reflect.Func || ftyp.NumIn() != 3 || ftyp.NumOut() != 0 || ftyp.In(0) != testingT || ftyp.In(1) != stringT {
		panic(fmt.Sprintf("bad test function type: want func(*testing.T, string, <TestType>), have %s", ftyp))
	}
	testType := ftyp.In(2)
	mp := reflect.New(testType)
	return mp.Elem()
}

func runTestFunc(runTest interface{}, t *testing.T, name string, m reflect.Value) {
	reflect.ValueOf(runTest).Call([]reflect.Value{
		reflect.ValueOf(t),
		reflect.ValueOf(name),
		m,
	})
}
