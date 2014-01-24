package rp2cgo2

import (
	"github.com/nwidger/m65go2"
)

type Registers struct {
	Controller uint8
	Mask       uint8
	Status     uint8
	OAMAddress uint8
	OAMData    uint8
	Scroll     uint8
	Address    uint8
	Data       uint8
}

func (reg *Registers) Reset() {
	reg.Controller = 0x00
	reg.Mask = 0x00
	reg.Status = 0x00
	reg.OAMAddress = 0x00
	reg.OAMData = 0x00
	reg.Scroll = 0x00
	reg.Address = 0x00
	reg.Data = 0x00
}

type RP2C02 struct {
	clock     *m65go2.Divider
	Registers Registers
}

func NewRP2C02(clock m65go2.Clocker, divisor uint64) *RP2C02 {
	divider := m65go2.NewDivider(clock, divisor)
	return &RP2C02{clock: divider}
}

func (ppu *RP2C02) Reset() {
	ppu.Registers.Reset()
}

func (ppu *RP2C02) Mappings() (fetch, store []uint16) {
	fetch = []uint16{}
	store = []uint16{}

	for i := uint16(0x2000); i <= 0x3fff; i++ {
		switch i & 0x0007 {
		case 0x0000:
			store = append(store, i)
		case 0x0001:
			store = append(store, i)
		case 0x0002:
			fetch = append(fetch, i)
		case 0x0003:
			store = append(store, i)
		case 0x0004:
			fetch = append(fetch, i)
			store = append(store, i)
		case 0x0005:
			store = append(store, i)
		case 0x0006:
			store = append(store, i)
		case 0x0007:
			fetch = append(fetch, i)
			store = append(store, i)
		}
	}

	return
}

func (ppu *RP2C02) Fetch(address uint16) (value uint8) {
	switch {
	case address >= 0x2000 && address <= 0x3fff:
		index := address & 0x0007
		switch index {
		// Status
		case 0x0002:
			value = ppu.Registers.Status
		// OAMData
		case 0x0004:
			value = ppu.Registers.OAMData
		// Data
		case 0x0007:
			value = ppu.Registers.Data
		}
	}

	return
}

func (ppu *RP2C02) Store(address uint16, value uint8) (oldValue uint8) {
	switch {
	case address >= 0x2000 && address <= 0x3fff:
		index := address & 0x0007

		switch index {
		// Controller
		case 0x0000:
			oldValue = ppu.Registers.Controller
			ppu.Registers.Controller = value
		// Mask
		case 0x0001:
			oldValue = ppu.Registers.Mask
			ppu.Registers.Mask = value
		// OAMAddress
		case 0x0003:
			oldValue = ppu.Registers.OAMAddress
			ppu.Registers.OAMAddress = value
		// OAMData
		case 0x0004:
			oldValue = ppu.Registers.OAMData
			ppu.Registers.OAMData = value
		// Scroll
		case 0x0005:
			oldValue = ppu.Registers.Scroll
			ppu.Registers.Scroll = value
		// Address
		case 0x0006:
			oldValue = ppu.Registers.Address
			ppu.Registers.Address = value
		// Data
		case 0x0007:
			oldValue = ppu.Registers.Data
			ppu.Registers.Data = value

		}
	}

	return
}
