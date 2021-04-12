/*******************************************************************************
*   (c) 2018 ZondaX GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
********************************************************************************/

package ledger_cosmos_go

import (
	"fmt"
	"math"

	"github.com/zondax/ledger-go"
)

const (
	userCLA = 0x55

	userINSGetVersion                   = 0
	userINSPublicKeySECP256K1           = 1
	userINSSignSECP256K1                = 2
	userINSPublicKeySECP256K1ShowBech32 = 3

	userINSHash                   = 100
	userINSPublicKeySECP256K1Test = 101
	userINSSignSECP256K1Test      = 103

	userMessageChunkSize = 250
)

// LedgerCosmos represents a connection to the Cosmos app in a Ledger Nano S device
type LedgerCosmos struct {
	api     *ledger_go.Ledger
	version VersionInfo
}

// RequiredCosmosUserAppVersion indicates the minimum required version of the Cosmos app
func RequiredCosmosUserAppVersion() VersionInfo {
	return VersionInfo{Major: 1, Minor: 0,}
}

// FindLedgerCosmosUserApp finds a Cosmos user app running in a ledger device
func FindLedgerCosmosUserApp() (*LedgerCosmos, error) {
	ledgerAPI, err := ledger_go.FindLedger()

	if err != nil {
		return nil, err
	}

	app := LedgerCosmos{ledgerAPI, VersionInfo{}}
	appVersion, err := app.GetVersion()

	if err != nil {
		defer ledgerAPI.Close()
		if err.Error() == "[APDU_CODE_CLA_NOT_SUPPORTED] Class not supported" {
			return nil, fmt.Errorf("are you sure the Cosmos app is open?")
		}
		return nil, err
	}

	req := RequiredCosmosUserAppVersion()
	if !CheckVersion(*appVersion, req) {
		defer ledgerAPI.Close()
		return nil, fmt.Errorf(
			"version not supported. Required >v%d.%d.%d", req.Major, req.Minor, req.Patch)
	}

	return &app, err
}

// Close closes a connection with the Cosmos user app
func (ledger *LedgerCosmos) Close() error {
	return ledger.api.Close()
}

// GetVersion returns the current version of the Cosmos user app
func (ledger *LedgerCosmos) GetVersion() (*VersionInfo, error) {
	message := []byte{userCLA, userINSGetVersion, 0, 0, 0}
	response, err := ledger.api.Exchange(message)

	if err != nil {
		return nil, err
	}

	if len(response) < 4 {
		return nil, fmt.Errorf("invalid response")
	}

	ledger.version = VersionInfo{
		AppMode: response[0],
		Major:   response[1],
		Minor:   response[2],
		Patch:   response[3],
	}

	return &ledger.version, nil
}

// SignSECP256K1 signs a transaction using Cosmos user app
func (ledger *LedgerCosmos) SignSECP256K1(bip32Path []uint32, transaction []byte) ([]byte, error) {
	return ledger.sign(userINSSignSECP256K1, bip32Path, transaction)
}

// GetPublicKeySECP256K1 retrieves the public key for the corresponding bip32 derivation path
func (ledger *LedgerCosmos) GetPublicKeySECP256K1(bip32Path []uint32) ([]byte, error) {
	pathBytes, err := GetBip32bytes(bip32Path, 3)
	if err != nil {
		return nil, err
	}
	header := []byte{userCLA, userINSPublicKeySECP256K1, 0, 0, byte(len(pathBytes))}
	message := append(header, pathBytes...)

	response, err := ledger.api.Exchange(message)

	if err != nil {
		return nil, err
	}

	if len(response) < 4 {
		return nil, fmt.Errorf("invalid response")
	}

	return response, nil
}

func validHRPByte(b byte) bool {
	// https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki
	return b >= 33 && b <= 126
}

// ShowAddressSECP256K1 shows the address for the corresponding bip32 derivation path
func (ledger *LedgerCosmos) ShowAddressSECP256K1(bip32Path []uint32, hrp string) error {
	if len(hrp) > 83 {
		return fmt.Errorf("hrp len should be <10")
	}

	hrpBytes := []byte(hrp)
	for _, b := range hrpBytes {
		if !validHRPByte(b) {
			return fmt.Errorf("all characters in the HRP must be in the [33, 126] range")
		}
	}

	// Check that app is at least 1.1.0
	requiredVersion := VersionInfo{0, 1, 1, 0,}
	if !CheckVersion(ledger.version, requiredVersion){
		return fmt.Errorf("command requires at least app version %v", requiredVersion)
	}

	pathBytes, err := GetBip32bytes(bip32Path, 3)
	if err != nil {
		return err
	}

	// Prepare message
	header := []byte{userCLA, userINSPublicKeySECP256K1ShowBech32, 0, 0, 0}
	message := append(header, byte(len(hrpBytes)))
	message = append(message, hrpBytes...)
	message = append(message, pathBytes...)
	message[4] = byte(len(message) - len(header)) // update length

	_, err = ledger.api.Exchange(message)

	return err
}

// Hash returns the hash for the transaction (only enabled in test mode apps)
func (ledger *LedgerCosmos) Hash(transaction []byte) ([]byte, error) {

	var packetIndex = byte(1)
	var packetCount = byte(math.Ceil(float64(len(transaction)) / float64(userMessageChunkSize)))

	var finalResponse []byte
	for packetIndex <= packetCount {
		chunk := userMessageChunkSize
		if len(transaction) < userMessageChunkSize {
			chunk = len(transaction)
		}

		header := []byte{userCLA, userINSHash, packetIndex, packetCount, byte(chunk)}
		message := append(header, transaction[:chunk]...)
		response, err := ledger.api.Exchange(message)

		if err != nil {
			return nil, err
		}
		finalResponse = response
		packetIndex++
		transaction = transaction[chunk:]
	}
	return finalResponse, nil
}

// TestGetPublicKeySECP256K1 (only enabled in test mode apps)
func (ledger *LedgerCosmos) TestGetPublicKeySECP256K1() ([]byte, error) {
	message := []byte{userCLA, userINSPublicKeySECP256K1Test, 0, 0, 0}
	response, err := ledger.api.Exchange(message)

	if err != nil {
		return nil, err
	}

	if len(response) < 4 {
		return nil, fmt.Errorf("invalid response")
	}

	return response, nil
}

// TestSignSECP256K1 (only enabled in test mode apps)
func (ledger *LedgerCosmos) TestSignSECP256K1(transaction []byte) ([]byte, error) {
	var packetIndex byte = 1
	var packetCount = byte(math.Ceil(float64(len(transaction)) / float64(userMessageChunkSize)))

	var finalResponse []byte

	for packetIndex <= packetCount {

		chunk := userMessageChunkSize
		if len(transaction) < userMessageChunkSize {
			chunk = len(transaction)
		}

		header := []byte{userCLA, userINSSignSECP256K1Test, packetIndex, packetCount, byte(chunk)}
		message := append(header, transaction[:chunk]...)

		response, err := ledger.api.Exchange(message)

		if err != nil {
			return nil, err
		}

		finalResponse = response
		packetIndex++
		transaction = transaction[chunk:]
	}
	return finalResponse, nil
}

func (ledger *LedgerCosmos) sign(instruction byte, bip32Path []uint32, transaction []byte) ([]byte, error) {
	var packetIndex byte = 1
	var packetCount = 1 + byte(math.Ceil(float64(len(transaction))/float64(userMessageChunkSize)))

	var finalResponse []byte

	var message []byte

	for packetIndex <= packetCount {
		chunk := userMessageChunkSize
		if packetIndex == 1 {
			pathBytes, err := GetBip32bytes(bip32Path, 3)
			if err != nil {
				return nil, err
			}
			header := []byte{userCLA, instruction, packetIndex, packetCount, byte(len(pathBytes))}
			message = append(header, pathBytes...)
		} else {
			if len(transaction) < userMessageChunkSize {
				chunk = len(transaction)
			}
			header := []byte{userCLA, instruction, packetIndex, packetCount, byte(chunk)}
			message = append(header, transaction[:chunk]...)
		}

		response, err := ledger.api.Exchange(message)
		if err != nil {
			if err.Error() == "[APDU_CODE_BAD_KEY_HANDLE] The parameters in the data field are incorrect" {
				// In this special case, we can extract additional info
				errorMsg := string(response)
				switch errorMsg {
				case "ERROR: JSMN_ERROR_NOMEM":
					return nil, fmt.Errorf("Not enough tokens were provided");
				case "PARSER ERROR: JSMN_ERROR_INVAL":
					return nil, fmt.Errorf("Unexpected character in JSON string");
				case "PARSER ERROR: JSMN_ERROR_PART":
					return nil, fmt.Errorf("The JSON string is not a complete.");
				}
				return nil, fmt.Errorf(errorMsg)
			}
			return nil, err
		}

		finalResponse = response
		if packetIndex > 1 {
			transaction = transaction[chunk:]
		}
		packetIndex++

	}
	return finalResponse, nil
}
