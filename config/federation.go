package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
)

func ExportFederationFile(fedFile string, config *Config) error {
	buf := new(bytes.Buffer)

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config.Federation); err != nil {
		return err
	}

	return ioutil.WriteFile(fedFile, buf.Bytes(), 0644)
}

func LoadFederationFile(fedFile string, config *Config) error {
	file, err := os.Open(fedFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(config.Federation)
}
