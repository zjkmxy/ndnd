/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2024 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package face

import (
	"fmt"
	"strconv"

	"github.com/named-data/YaNFD/core"
	"github.com/named-data/YaNFD/ndn"
	"tinygo.org/x/bluetooth"
)

const BLEMTU = 512 // See esp8266ndn, it uses 512 & 517
// Note that this MTU does not match the real MTU: tinygo/bluetooth only supports the default 23
// However, it can receive and send packet with 512B

var (
	serviceUUID = bluetooth.NewUUID([16]byte{0x09, 0x95, 0x77, 0xe3, 0x07, 0x88, 0x41, 0x2a, 0x88, 0x24, 0x39, 0x50, 0x84, 0xd9, 0x73, 0x91})
	csUUID      = bluetooth.NewUUID([16]byte{0xcc, 0x5a, 0xbb, 0x89, 0xa5, 0x41, 0x46, 0xd8, 0xa3, 0x51, 0x2f, 0x95, 0xa6, 0xa8, 0x1f, 0x49})
	scUUID      = bluetooth.NewUUID([16]byte{0x97, 0x2f, 0x95, 0x27, 0x0d, 0x83, 0x42, 0x61, 0xb9, 0x5d, 0xb1, 0xb2, 0xfc, 0x73, 0xbd, 0xe4})
)

// BLEPeripheral is a Bluetooth Low Energy GATT transport, running as a peripheral.
type BLEPeripheral struct {
	transportBase

	cs  bluetooth.Characteristic
	sc  bluetooth.Characteristic
	adv *bluetooth.Advertisement
}

var _ transport = &BLEPeripheral{} // trait

// NewBLEPeripheral creates a BLE transport.
func NewBLEPeripheral(localName string) (t *BLEPeripheral, err error) {
	remoteURI := ndn.MakeBLEURI(localName, serviceUUID.String())
	t = &BLEPeripheral{
		cs: bluetooth.Characteristic{},
		sc: bluetooth.Characteristic{},
	}

	adapter := bluetooth.DefaultAdapter
	err = adapter.Enable()
	if err != nil {
		core.LogError(nil, "Unable to start BLE adaptor: ", err)
		t = nil
		return
	}
	t.adv = adapter.DefaultAdvertisement()
	err = t.adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    localName,
		ServiceUUIDs: []bluetooth.UUID{serviceUUID},
	})
	if err != nil {
		core.LogError(nil, "Unable to start BLE advertiser: ", err)
		t = nil
		return
	}
	err = t.adv.Start()
	if err != nil {
		core.LogError(nil, "Unable to start BLE advertiser: ", err)
		t = nil
		return
	}

	err = adapter.AddService(&bluetooth.Service{
		UUID: serviceUUID,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &t.cs,
				UUID:   csUUID,
				Flags: (bluetooth.CharacteristicWritePermission |
					bluetooth.CharacteristicWriteWithoutResponsePermission |
					bluetooth.CharacteristicNotifyPermission),
				WriteEvent: t.onWrite,
			},
			{
				Handle: &t.sc,
				UUID:   scUUID,
				Flags:  bluetooth.CharacteristicNotifyPermission | bluetooth.CharacteristicReadPermission,
			},
		},
	})
	if err != nil {
		core.LogError(nil, "Unable to start BLE service: ", err)
		t = nil
		return
	}

	scope := ndn.NonLocal
	t.makeTransportBase(remoteURI, remoteURI, PersistencyPermanent, scope, ndn.MultiAccess, BLEMTU)
	t.changeState(ndn.Up)
	return
}

func (t *BLEPeripheral) String() string {
	return "BLEPeripheral, FaceID=" + strconv.FormatUint(t.faceID, 10)
}

// SetPersistency changes the persistency of the face.
func (t *BLEPeripheral) SetPersistency(persistency Persistency) bool {
	return persistency == PersistencyPermanent
}

// GetSendQueueSize returns the current size of the send queue.
func (t *BLEPeripheral) GetSendQueueSize() uint64 {
	return 0
}

func (t *BLEPeripheral) sendFrame(frame []byte) {
	_, err := t.sc.Write(frame)
	if err != nil {
		core.LogError(t, "Unable to write BLE frame", err)
	}

	fmt.Println("SENT:", frame)

	t.nOutBytes += uint64(len(frame))
}

func (t *BLEPeripheral) changeState(new ndn.State) {
	if t.state == new {
		return
	}

	core.LogInfo(t, "state: ", t.state, " -> ", new)
	t.state = new

	if t.state != ndn.Up {
		core.LogInfo(t, "Closing BLE socket")
		t.hasQuit <- true
		t.adv.Stop()

		// Stop link service
		t.linkService.tellTransportQuit()

		FaceTable.Remove(t.faceID)
	}
}

func (t *BLEPeripheral) onWrite(client bluetooth.Connection, offset int, value []byte) {
	fmt.Println("RECVED:", value)

	t.linkService.handleIncomingFrame(value)
	t.nInBytes += uint64(len(value))
}
