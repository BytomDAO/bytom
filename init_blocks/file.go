package initblocks

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// ReadFileLines read file all lines
func ReadFileLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lines = append(lines, line)
	}
	return lines, nil
}

// ReadWholeFile read whole file
func ReadWholeFile(filePath string) (string, error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// WriteToFileByLines write strings as lines to file
func WriteToFileByLines(fileName string, strs []string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}

	defer file.Close()

	for _, str := range strs {
		line := fmt.Sprintf("%s\n", str)
		if _, err := file.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

func WriteFile(filename, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, data)
	return err
}
