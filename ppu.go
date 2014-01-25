package rp2cgo2

import (
	"github.com/nwidger/m65go2"
)

type ControllerFlag uint8

const (
	BaseNametableAddress ControllerFlag = 1 << iota
	_
	VRAMAddressIncrement
	SpritePatternAddress
	BackgroundPatternAddress
	SpriteSize
	_
	NMIOnVBlank
)

type MaskFlag uint8

const (
	Grayscale MaskFlag = 1 << iota
	ShowBackgroundLeft
	ShowSpritesLeft
	ShowBackground
	ShowSprites
	IntensifyReds
	IntensifyGreens
	IntensifyBlues
)

type StatusFlag uint8

const (
	_ StatusFlag = 1 << iota
	_
	_
	_
	_
	SpriteOverflow
	Sprite0Hit
	VBlankStarted
)

type Registers struct {
	Controller uint8
	Mask       uint8
	Status     uint8
	OAMAddress uint8
	OAMData    uint8
	Scroll     uint16
	Address    uint16
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
	clock        *m65go2.Divider
	latch        bool
	latchAddress uint16
	Registers    Registers
}

func NewRP2C02(clock m65go2.Clocker, divisor uint64) *RP2C02 {
	divider := m65go2.NewDivider(clock, divisor)
	return &RP2C02{clock: divider}
}

func (ppu *RP2C02) Reset() {
	ppu.latch = false
	ppu.Registers.Reset()
}

func (ppu *RP2C02) controller(flag ControllerFlag) (value uint16) {
	byte := ppu.Registers.Controller
	bit := (byte >> flag) & 0x01

	switch flag {
	case BaseNametableAddress:
		// 0x2000 | 0x2400 | 0x2800 | 0x2c00
		value = 0x2000 + (uint16(byte&0x03) * 0x0400)
	case VRAMAddressIncrement:
		switch bit {
		case 0:
			value = 1
		case 1:
			value = 32
		}
	case SpritePatternAddress:
		// 0x0000 | 0x1000
		switch bit {
		case 0:
			value = 0x0000
		case 1:
			value = 0x1000
		}
	case BackgroundPatternAddress:
		// 0x0000 | 0x1000
		switch bit {
		case 0:
			value = 0x0000
		case 1:
			value = 0x1000
		}

	case SpriteSize:
		// 8x8 | 8x16
		switch bit {
		case 0:
			value = 8
		case 1:
			value = 16
		}
	case NMIOnVBlank:
		switch bit {
		case 0:
			value = 0
		case 1:
			value = 1
		}
	}

	return
}

func (ppu *RP2C02) mask(flag MaskFlag) (value bool) {
	if (ppu.Registers.Mask>>flag)&0x01 != 0 {
		value = true
	}

	return
}

func (ppu *RP2C02) status(flag StatusFlag) (value bool) {
	if (ppu.Registers.Mask>>flag)&0x01 != 0 {
		value = true
	}

	return
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
			ppu.Registers.Status &^= uint8(VBlankStarted)
			ppu.latch = false
			ppu.latchAddress = 0x0000
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
			ppu.latchAddress = (ppu.latchAddress & 0x73ff) | uint16(ppu.controller(BaseNametableAddress))<<10
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
			if !ppu.latch {
				// Horizontal scroll offset
				// 0x7fe0 == 0111 1111 1110 0000b

				// copy upper 5 bits of value into latchAddress
				// copy lower 3 bits of value into Scroll
				ppu.latchAddress = (ppu.latchAddress & 0x7fe0) | uint16(value>>3)
				ppu.Registers.Scroll = uint16(value & 0x07)
			} else {
				// Vertical scroll offset
				// 0x0c1f == 0000 1100 0001 1111b
				// 0x73e0 == 0111 0011 1110 0000b
				ppu.latchAddress = (ppu.latchAddress & 0x0c1f) | ((uint16(value)<<2 | uint16(value)<<12) & 0x73e0)
			}

			ppu.latch = !ppu.latch
		// Address
		case 0x0006:
			if !ppu.latch {
				ppu.latchAddress = (ppu.latchAddress & 0x00ff) | uint16(value&0x3f)<<8
			} else {
				ppu.latchAddress = (ppu.latchAddress & 0x7f00) | uint16(value)
				ppu.Registers.Address = ppu.latchAddress
			}

			ppu.latch = !ppu.latch
		// Data
		case 0x0007:
			oldValue = ppu.Registers.Data
			ppu.Registers.Data = value

			ppu.Registers.Address += ppu.controller(VRAMAddressIncrement)

		}
	}

	return
}
