package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
)

// TrapSignal catches the SIGTERM and executes cb function. After that it exits
// with code 1.
func TrapSignal(cb func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			fmt.Printf("captured %v, exiting...\n", sig)
			if cb != nil {
				cb()
			}
			os.Exit(1)
		}
	}()
	select {}
}

func Exit(s string) {
	fmt.Printf(s + "\n")
	os.Exit(1)
}

func EnsureDir(dir string, mode os.FileMode) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, mode)
		if err != nil {
			return fmt.Errorf("Could not create directory %v. %v", dir, err)
		}
	}
	return nil
}

func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func WriteFile(filePath string, contents []byte, mode os.FileMode) error {
	return ioutil.WriteFile(filePath, contents, mode)
}

func MustWriteFile(filePath string, contents []byte, mode os.FileMode) {
	err := WriteFile(filePath, contents, mode)
	if err != nil {
		Exit(Fmt("MustWriteFile failed: %v", err))
	}
}
