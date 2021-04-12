// +build !ledger_mock,!ledger_zemu

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

package ledger_go

import (
	"errors"
	"fmt"
	"sync"

	"github.com/zondax/hid"
)

const (
	VendorLedger         = 0x2c97
	UsagePageLedgerNanoS = 0xffa0
	//ProductNano     = 1
	Channel    = 0x0101
	PacketSize = 64
)

type LedgerAdminHID struct{}

type LedgerDeviceHID struct {
	device      *hid.Device
	readCo      *sync.Once
	readChannel chan []byte
}

func NewLedgerAdmin() *LedgerAdminHID {
	return &LedgerAdminHID{}
}

func (admin *LedgerAdminHID) ListDevices() ([]string, error) {
	devices := hid.Enumerate(0, 0)

	if len(devices) == 0 {
		fmt.Printf("No devices")
	}

	for _, d := range devices {
		fmt.Printf("============ %s\n", d.Path)
		fmt.Printf("VendorID      : %x\n", d.VendorID)
		fmt.Printf("ProductID     : %x\n", d.ProductID)
		fmt.Printf("Release       : %x\n", d.Release)
		fmt.Printf("Serial        : %x\n", d.Serial)
		fmt.Printf("Manufacturer  : %s\n", d.Manufacturer)
		fmt.Printf("Product       : %s\n", d.Product)
		fmt.Printf("UsagePage     : %x\n", d.UsagePage)
		fmt.Printf("Usage         : %x\n", d.Usage)
		fmt.Printf("\n")
	}

	return []string{}, nil
}

func isLedgerDevice(d hid.DeviceInfo) bool {
	deviceFound := d.UsagePage == UsagePageLedgerNanoS
	// Workarounds for possible empty usage pages
	return deviceFound ||
		(d.Product == "Nano S" && d.Interface == 0) ||
		(d.Product == "Nano X" && d.Interface == 0)
}

func (admin *LedgerAdminHID) CountDevices() int {
	devices := hid.Enumerate(0, 0)

	count := 0
	for _, d := range devices {
		if isLedgerDevice(d) {
			count++
		}
	}

	return count
}

func newDevice(dev *hid.Device) *LedgerDeviceHID {
	return &LedgerDeviceHID{
		device: dev,
		readCo: new(sync.Once),
		readChannel: make(chan []byte),
	}
}

func (admin *LedgerAdminHID) Connect(requiredIndex int) (LedgerDevice, error) {
	devices := hid.Enumerate(VendorLedger, 0)

	currentIndex := 0
	for _, d := range devices {
		if isLedgerDevice(d) {
			if currentIndex == requiredIndex {
				device, err := d.Open()
				if err != nil {
					return nil, err
				}
				deviceHID := newDevice(device)
				return deviceHID, nil
			}
			currentIndex++
			if currentIndex > requiredIndex {
				break
			}
		}
	}

	return nil, fmt.Errorf("LedgerHID device (idx %d) not found", requiredIndex)
}

func (ledger *LedgerDeviceHID) write(buffer []byte) (int, error) {
	totalBytes := len(buffer)
	totalWrittenBytes := 0
	for totalBytes > totalWrittenBytes {
		writtenBytes, err := ledger.device.Write(buffer)

		if err != nil {
			return totalWrittenBytes, err
		}
		buffer = buffer[writtenBytes:]
		totalWrittenBytes += writtenBytes
	}
	return totalWrittenBytes, nil
}

func (ledger *LedgerDeviceHID) Read() <-chan []byte {
	ledger.readCo.Do(ledger.initReadChannel)
	return ledger.readChannel
}

func (ledger *LedgerDeviceHID) initReadChannel() {
	ledger.readChannel = make(chan []byte, 30)
	go ledger.readThread()
}

func (ledger *LedgerDeviceHID) readThread() {
	defer close(ledger.readChannel)

	for {
		buffer := make([]byte, PacketSize)
		readBytes, err := ledger.device.Read(buffer)

		if err != nil {
			return
		}
		select {
		case ledger.readChannel <- buffer[:readBytes]:
		default:
		}
	}
}

func (ledger *LedgerDeviceHID) Exchange(command []byte) ([]byte, error) {
	if len(command) < 5 {
		return nil, fmt.Errorf("APDU commands should not be smaller than 5")
	}

	if (byte)(len(command)-5) != command[4] {
		return nil, fmt.Errorf("APDU[data length] mismatch")
	}

	serializedCommand, err := WrapCommandAPDU(Channel, command, PacketSize)
	if err != nil {
		return nil, err
	}

	// Write all the packets
	_, err = ledger.write(serializedCommand)
	if err != nil {
		return nil, err
	}

	readChannel := ledger.Read()

	response, err := UnwrapResponseAPDU(Channel, readChannel, PacketSize)
	if err != nil {
		return nil, err
	}

	if len(response) < 2 {
		return nil, fmt.Errorf("len(response) < 2")
	}

	swOffset := len(response) - 2
	sw := codec.Uint16(response[swOffset:])

	if sw != 0x9000 {
		return response[:swOffset], errors.New(ErrorMessage(sw))
	}

	return response[:swOffset], nil
}

func (ledger *LedgerDeviceHID) Close() error {
	return ledger.device.Close()
}
