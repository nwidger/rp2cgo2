package rp2cgo2

import (
	"github.com/nwidger/m65go2"
	"testing"
	"time"
)

func TestController(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor)

	for i := uint16(0x2000); i <= 0x3fff; i += 0x0008 {
		ppu.Registers.Controller = 0x00

		value := uint8(i % 0xff)

		ppu.Store(i, value)

		if ppu.Registers.Controller != value {
			t.Errorf("Register is %02X not %02X\n", ppu.Registers.Controller, value)
		}
	}

	// BaseNametableAddress
	ppu.Registers.Controller = 0xff - 0x03

	if ppu.controller(BaseNametableAddress) != 0x2000 {
		t.Error("BaseNametableAddress is %04X not 0x2000", ppu.controller(BaseNametableAddress))
	}

	ppu.Registers.Controller = 0xff - 0x02

	if ppu.controller(BaseNametableAddress) != 0x2400 {
		t.Error("BaseNametableAddress is %04X not 0x2400", ppu.controller(BaseNametableAddress))
	}

	ppu.Registers.Controller = 0xff - 0x01

	if ppu.controller(BaseNametableAddress) != 0x2800 {
		t.Error("BaseNametableAddress is %04X not 0x2800", ppu.controller(BaseNametableAddress))
	}

	ppu.Registers.Controller = 0xff

	if ppu.controller(BaseNametableAddress) != 0x2c00 {
		t.Error("BaseNametableAddress is %04X not 0x2c00", ppu.controller(BaseNametableAddress))
	}

	// VRAMAddressIncrement
	ppu.Registers.Controller = ^uint8(VRAMAddressIncrement)

	if ppu.controller(VRAMAddressIncrement) != 1 {
		t.Error("VRAMAddressIncrement is not 1")
	}

	ppu.Registers.Controller = uint8(VRAMAddressIncrement)

	if ppu.controller(VRAMAddressIncrement) != 32 {
		t.Error("VRAMAddressIncrement is not 32")
	}

	// SpritePatternAddress
	ppu.Registers.Controller = ^uint8(SpritePatternAddress)

	if ppu.controller(SpritePatternAddress) != 0x0000 {
		t.Error("SpritePatternAddress is not 0x0000")
	}

	ppu.Registers.Controller = uint8(SpritePatternAddress)

	if ppu.controller(SpritePatternAddress) != 0x1000 {
		t.Error("SpritePatternAddress is not 0x1000")
	}

	// BackgroundPatternAddress
	ppu.Registers.Controller = ^uint8(BackgroundPatternAddress)

	if ppu.controller(BackgroundPatternAddress) != 0x0000 {
		t.Error("BackgroundPatternAddress is not 0x0000")
	}

	ppu.Registers.Controller = uint8(BackgroundPatternAddress)

	if ppu.controller(BackgroundPatternAddress) != 0x1000 {
		t.Error("BackgroundPatternAddress is not 0x1000")
	}

	// SpriteSize
	ppu.Registers.Controller = ^uint8(SpriteSize)

	if ppu.controller(SpriteSize) != 8 {
		t.Error("SpriteSize is not 8")
	}

	ppu.Registers.Controller = uint8(SpriteSize)

	if ppu.controller(SpriteSize) != 16 {
		t.Error("SpriteSize is not 16")
	}

	// NMIOnVBlank
	ppu.Registers.Controller = ^uint8(NMIOnVBlank)

	if ppu.controller(NMIOnVBlank) != 0 {
		t.Error("NMIOnVBlank is not 0")
	}

	ppu.Registers.Controller = uint8(NMIOnVBlank)

	if ppu.controller(NMIOnVBlank) != 1 {
		t.Error("NMIOnVBlank is not 1")
	}
}

func TestMask(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor)

	for i := uint16(0x2001); i <= 0x3fff; i += 0x0008 {
		ppu.Registers.Mask = 0x00

		value := uint8(i % 0xff)

		ppu.Store(i, value)

		if ppu.Registers.Mask != value {
			t.Errorf("Register is %02X not %02X\n", ppu.Registers.Mask, value)
		}
	}

	for _, m := range []MaskFlag{
		Grayscale, ShowBackgroundLeft, ShowSpritesLeft, ShowBackground,
		ShowSprites, IntensifyReds, IntensifyGreens, IntensifyBlues,
	} {
		ppu.Registers.Mask = uint8(m)

		if !ppu.mask(m) {
			t.Errorf("Mask %v is not true", m)
		}
	}
}

func TestStatus(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor)

	for i := uint16(0x2002); i <= 0x3fff; i += 0x0008 {
		ppu.Registers.Status = 0x00
		value := uint8(i % 0xff)
		ppu.Registers.Status = value

		if ppu.Fetch(i) != value {
			t.Errorf("Memory is %02X not %02X\n", ppu.Fetch(i), value)
		}
	}

	ppu.Registers.Status = 0xff
	ppu.latch = true

	if ppu.Fetch(0x2002) != 0xff {
		t.Errorf("Memory is %02X not 0xff\n", ppu.Fetch(0x2002))
	}

	if ppu.Registers.Status != 0x7f {
		t.Error("VBlankStarted flag is set")
	}

	if ppu.latch {
		t.Error("Latch is true")
	}

	for _, s := range []StatusFlag{
		SpriteOverflow,
		Sprite0Hit,
		VBlankStarted,
	} {
		ppu.Registers.Status = uint8(s)

		if !ppu.status(s) {
			t.Errorf("Status %v is not true", s)
		}
	}
}
