//go:build !linux

package face

import "errors"

func NewBLEPeripheral(localName string) (t transport, err error) {
	return nil, errors.New("BLE is not supported on this platform")
}
