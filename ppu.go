package rp2cgo2

import (
	"fmt"
	"github.com/nwidger/m65go2"
	"github.com/nwidger/rp2ago3"
)

type Mirroring uint8

const (
	Horizontal Mirroring = iota
	Vertical
	FourScreen
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

const (
	CYCLES_PER_SCANLINE uint64 = 341
	NUM_SCANLINES              = 262
	POWERUP_SCANLINE    uint16 = 241
)

type RP2C02 struct {
	clock        *m65go2.Divider
	latch        bool
	latchAddress uint16
	Registers    Registers
	Memory       *rp2ago3.MappedMemory
	Interrupt    func(state bool)
	scanline     uint16
	cycle        uint64
}

func NewRP2C02(clock m65go2.Clocker, divisor uint64, interrupt func(bool), mirroring Mirroring) *RP2C02 {
	divider := m65go2.NewDivider(clock, divisor)

	mem := rp2ago3.NewMappedMemory(m65go2.NewBasicMemory())
	mirrors := make(map[uint16]uint16)

	switch mirroring {
	case Horizontal:
		// Mirror nametable #1 to #0
		for i := uint16(0x2400); i <= 0x27ff; i++ {
			mirrors[i] = i - 0x0400
		}

		// Mirror nametable #3 to #2
		for i := uint16(0x2c00); i <= 0x2fff; i++ {
			mirrors[i] = i - 0x0400
		}
	case Vertical:
		// Mirror nametable #2 to #0
		for i := uint16(0x2800); i <= 0x2bff; i++ {
			mirrors[i] = i - 0x0800
		}

		// Mirror nametable #3 to #1
		for i := uint16(0x2c00); i <= 0x2fff; i++ {
			mirrors[i] = i - 0x0800
		}
	}

	// Mirrored nametables
	for i := uint16(0x3000); i <= 0x3eff; i++ {
		mirrors[i] = i - 0x1000
	}

	// Mirrored palette
	for _, i := range []uint16{0x3f10, 0x3f14, 0x3f18, 0x3f1c} {
		mirrors[i] = i - 0x0010
	}

	for i := uint16(0x3f20); i <= 0x3fff; i++ {
		mirrors[i] = i - 0x0020
	}

	mem.AddMirrors(mirrors)

	return &RP2C02{
		clock:     divider,
		Memory:    mem,
		Interrupt: interrupt,
	}
}

func (ppu *RP2C02) String() string {
	return fmt.Sprintf("CYC:%3d SL:%3d", ppu.cycle, ppu.scanline)
}

func (ppu *RP2C02) Reset() {
	ppu.latch = false
	ppu.Registers.Reset()
	ppu.Memory.Reset()
}

func (ppu *RP2C02) controller(flag ControllerFlag) (value uint16) {
	byte := ppu.Registers.Controller
	bit := byte & uint8(flag)

	switch flag {
	case BaseNametableAddress:
		// 0x2000 | 0x2400 | 0x2800 | 0x2c00
		value = 0x2000 + (uint16(byte&0x03) * 0x0400)
	case VRAMAddressIncrement:
		switch bit {
		case 0:
			value = 1
		default:
			value = 32
		}
	case SpritePatternAddress:
		// 0x0000 | 0x1000
		switch bit {
		case 0:
			value = 0x0000
		default:
			value = 0x1000
		}
	case BackgroundPatternAddress:
		// 0x0000 | 0x1000
		switch bit {
		case 0:
			value = 0x0000
		default:
			value = 0x1000
		}

	case SpriteSize:
		// 8x8 | 8x16
		switch bit {
		case 0:
			value = 8
		default:
			value = 16
		}
	case NMIOnVBlank:
		switch bit {
		case 0:
			value = 0
		default:
			value = 1
		}
	}

	return
}

func (ppu *RP2C02) mask(flag MaskFlag) (value bool) {
	if ppu.Registers.Mask&uint8(flag) != 0 {
		value = true
	}

	return
}

func (ppu *RP2C02) status(flag StatusFlag) (value bool) {
	if ppu.Registers.Status&uint8(flag) != 0 {
		value = true
	}

	return
}

func (ppu *RP2C02) Mappings(which rp2ago3.Mapping) (fetch, store []uint16) {
	fetch = []uint16{}
	store = []uint16{}

	switch which {
	case rp2ago3.CPU:
		for i := uint16(0x2000); i <= 0x2007; i++ {
			switch i {
			case 0x2000:
				store = append(store, i)
			case 0x2001:
				store = append(store, i)
			case 0x2002:
				fetch = append(fetch, i)
			case 0x2003:
				store = append(store, i)
			case 0x2004:
				fetch = append(fetch, i)
				store = append(store, i)
			case 0x2005:
				store = append(store, i)
			case 0x2006:
				store = append(store, i)
			case 0x2007:
				fetch = append(fetch, i)
				store = append(store, i)
			}
		}
	}

	return
}

func (ppu *RP2C02) Fetch(address uint16) (value uint8) {
	switch address {
	// Status
	case 0x2002:
		value = ppu.Registers.Status

		ppu.Registers.Status &^= uint8(VBlankStarted)
		ppu.latch = false
	// OAMData
	case 0x2004:
		value = ppu.Registers.OAMData
	// Data
	case 0x2007:
		value = ppu.Registers.Data

		vramAddress := ppu.Registers.Address & 0x3fff
		ppu.Registers.Data = ppu.Memory.Fetch(vramAddress)

		if vramAddress >= 0x3f00 {
			value = ppu.Registers.Data
		}

		ppu.Registers.Address += ppu.controller(VRAMAddressIncrement)
	}

	return
}

func (ppu *RP2C02) Store(address uint16, value uint8) (oldValue uint8) {
	switch address {
	// Controller
	case 0x2000:
		oldValue = ppu.Registers.Controller
		ppu.Registers.Controller = value
		ppu.latchAddress = (ppu.latchAddress & 0x73ff) | uint16(ppu.controller(BaseNametableAddress))<<10
	// Mask
	case 0x2001:
		oldValue = ppu.Registers.Mask
		ppu.Registers.Mask = value
	// OAMAddress
	case 0x2003:
		oldValue = ppu.Registers.OAMAddress
		ppu.Registers.OAMAddress = value
	// OAMData
	case 0x2004:
		oldValue = ppu.Registers.OAMData
		ppu.Registers.OAMData = value
	// Scroll
	case 0x2005:
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
	case 0x2006:
		if !ppu.latch {
			ppu.latchAddress = (ppu.latchAddress & 0x00ff) | uint16(value&0x3f)<<8
		} else {
			ppu.latchAddress = (ppu.latchAddress & 0x7f00) | uint16(value)
			ppu.Registers.Address = ppu.latchAddress
		}

		ppu.latch = !ppu.latch
	// Data
	case 0x2007:
		oldValue = ppu.Registers.Data
		ppu.Memory.Store(ppu.Registers.Address&0x3fff, value)
		ppu.Registers.Address += ppu.controller(VRAMAddressIncrement)
	}

	return
}

func (ppu *RP2C02) renderScanline() (cycles uint64) {
	ppu.cycle = 0
	ticks := ppu.clock.Ticks()
	cycles = CYCLES_PER_SCANLINE

	switch {
	// visible scanlines (0-239)
	case ppu.scanline >= 0 && ppu.scanline <= 239:
		// cycle 0

		ppu.cycle = 1
		ppu.clock.Await(ticks + ppu.cycle)

		// cycles 1-256

		ppu.cycle = 257
		ppu.clock.Await(ticks + ppu.cycle)

		// cycles 257-320

		ppu.cycle = 231
		ppu.clock.Await(ticks + ppu.cycle)

		// cycles 321-336

		ppu.cycle = 337
		ppu.clock.Await(ticks + ppu.cycle)

		// cycles 337-340

		ppu.cycle = 341
		ppu.clock.Await(ticks + ppu.cycle)
	// post-render ppu.scanline
	case ppu.scanline == 240:

	// vertical blanking scanlines
	case ppu.scanline == 241:
		ppu.cycle = 1
		ppu.clock.Await(ticks + ppu.cycle)

		ppu.Registers.Status |= uint8(VBlankStarted)

		if ppu.Interrupt != nil {
			ppu.Interrupt(true)
		}
	case ppu.scanline >= 242 && ppu.scanline <= 260:

	// pre-render ppu.scanline
	case ppu.scanline == 261:
		ppu.cycle = 1
		ppu.clock.Await(ticks + ppu.cycle)
		ppu.Registers.Status &^= uint8(VBlankStarted | Sprite0Hit | SpriteOverflow)
	}

	cycles -= ppu.cycle

	return
}

func (ppu *RP2C02) Run() (err error) {
	ppu.scanline = POWERUP_SCANLINE

	for {
		ticks := ppu.clock.Ticks()
		ppu.clock.Await(ticks + ppu.renderScanline())
		ppu.scanline = (ppu.scanline + 1) % NUM_SCANLINES
	}

	return
}
