package rotatelogs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

func ExampleForceNewFile () {
	logDir, err := ioutil.TempDir("", "rotatelogs_test")
	if err != nil {
		fmt.Println("could not create log directory ", err)
		return
	}
	logPath := fmt.Sprintf("%s/test.log", logDir)

	for i := 0; i < 2; i++ {
		writer, err := rotatelogs.New(logPath,
			rotatelogs.ForceNewFile(),
		)
		if err != nil {
			fmt.Println("Could not open log file ", err)
			return
		}

		n, err := writer.Write([]byte("test"))
		if err != nil || n != 4 {
			fmt.Println("Write failed ", err, " number written ", n)
			return
		}
		err = writer.Close()
		if err != nil {
			fmt.Println("Close failed ", err)
			return
		}
	}

	files, err := ioutil.ReadDir(logDir)
	if err != nil {
		fmt.Println("ReadDir failed ", err)
		return
	}
	for _, file := range files {
		fmt.Println(file.Name(), file.Size())
	}

	err = os.RemoveAll(logDir)
	if err != nil {
		fmt.Println("RemoveAll failed ", err)
		return
	}
	// OUTPUT:
	// test.log 4
	// test.log.1 4
}
