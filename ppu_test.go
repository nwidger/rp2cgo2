package rp2cgo2

import (
	"github.com/nwidger/m65go2"
	"testing"
	"time"
)

func TestController(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor, Vertical)

	ppu.Registers.Controller = 0x00
	ppu.Store(0x2000, 0xff)

	if ppu.Registers.Controller != 0xff {
		t.Errorf("Register is %02X not 0xff\n", ppu.Registers.Controller)
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
	ppu := NewRP2C02(clock, divisor, Vertical)

	ppu.Registers.Mask = 0x00
	value := uint8(0xff)
	ppu.Store(0x2001, value)

	if ppu.Registers.Mask != value {
		t.Errorf("Register is %02X not %02X\n", ppu.Registers.Mask, value)
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
	ppu := NewRP2C02(clock, divisor, Vertical)

	ppu.Registers.Status = 0x00
	value := uint8(0xff)
	ppu.Registers.Status = value

	if ppu.Fetch(0x2002) != value {
		t.Errorf("Memory is %02X not %02X\n", ppu.Fetch(0x2002), value)
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

func TestVerticalMirroring(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor, Vertical)

	// Mirror nametable #2 to #0
	for i := uint16(0x2800); i <= 0x2bff; i++ {
		ppu.Memory.Store(i-0x0800, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x0800, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x0800) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}

	// Mirror nametable #3 to #1
	for i := uint16(0x2c00); i <= 0x2fff; i++ {
		ppu.Memory.Store(i-0x0800, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x0800, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x0800) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}

	// Mirror nametable #2 to #0
	for i := uint16(0x3000); i <= 0x33ff; i++ {
		ppu.Memory.Store(i-0x1000, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x1000, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x1000) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}

	// Mirror nametable #3 to #1
	for i := uint16(0x3400); i <= 0x37ff; i++ {
		ppu.Memory.Store(i-0x1000, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x1000, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x1000) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}
}

func TestHorizontalMirroring(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor, Horizontal)

	// Mirror nametable #1 to #0
	for i := uint16(0x2400); i <= 0x27ff; i++ {
		ppu.Memory.Store(i-0x0400, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x0400, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x0400) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}

	// Mirror nametable #3 to #2
	for i := uint16(0x2c00); i <= 0x2fff; i++ {
		ppu.Memory.Store(i-0x0400, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x0400, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x0400) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}

	// Mirror nametable #1 to #0
	for i := uint16(0x3400); i <= 0x37ff; i++ {
		ppu.Memory.Store(i-0x1400, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x1400, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x1400) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}

	// Mirror nametable #3 to #2
	for i := uint16(0x3c00); i <= 0x3eff; i++ {
		ppu.Memory.Store(i-0x1400, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x1400, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x1400) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}
}

func TestPaletteMirroring(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor, Horizontal)

	// Mirrored palette
	for _, i := range []uint16{0x3f10, 0x3f14, 0x3f18, 0x3f1c} {
		ppu.Memory.Store(i-0x0010, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x0010, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x0010) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}

	for i := uint16(0x3f20); i <= 0x3fff; i++ {
		ppu.Memory.Store(i-0x0020, 0xff)

		if ppu.Memory.Fetch(i) != 0xff {
			t.Error("Memory is not 0xff")
		}

		ppu.Memory.Store(i-0x0020, 0x00)
		ppu.Memory.Store(i, 0xff)

		if ppu.Memory.Fetch(i-0x0020) != 0xff {
			t.Error("Memory is not 0xff")
		}
	}
}

func TestAddress(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor, Horizontal)

	ppu.Registers.Address = 0x0000
	ppu.Fetch(0x2002)

	ppu.Store(0x2006, 0x3f)
	ppu.Store(0x2006, 0xff)

	if ppu.Registers.Address != 0x3fff {
		t.Error("Register is not 0x3fff")
	}

	ppu.Fetch(0x2002)

	ppu.Store(0x2006, 0x01)
	ppu.Store(0x2006, 0x01)

	if ppu.Registers.Address != 0x0101 {
		t.Error("Register is not 0x0101")
	}
}

func TestDataIncrement1(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor, Horizontal)

	ppu.Registers.Address = 0x0000
	ppu.Fetch(0x2002)

	ppu.Store(0x2006, 0x01)
	ppu.Store(0x2006, 0x00)

	if ppu.Registers.Address != 0x0100 {
		t.Error("Register is not 0x0100")
	}

	ppu.Store(0x2007, 0xff)
	ppu.Store(0x2007, 0xff)
	ppu.Store(0x2007, 0xff)

	if ppu.Memory.Fetch(0x0100) != 0xff {
		t.Error("Memory is not 0xff")
	}

	if ppu.Memory.Fetch(0x0101) != 0xff {
		t.Error("Memory is not 0xff")
	}

	if ppu.Memory.Fetch(0x0102) != 0xff {
		t.Error("Memory is not 0xff")
	}

	if ppu.Registers.Address != 0x0103 {
		t.Error("Register is not 0x0103")
	}

	ppu.Memory.Store(0x0103, 0xff)
	ppu.Fetch(0x2007)

	if ppu.Fetch(0x2007) != 0xff {
		t.Error("Memory is not 0xff")
	}

	if ppu.Registers.Address != 0x0105 {
		t.Error("Register is not 0x0105")
	}

	ppu.Registers.Address = 0x0000
	ppu.Fetch(0x2002)

	ppu.Store(0x2006, 0x3f)
	ppu.Store(0x2006, 0x00)

	if ppu.Registers.Address != 0x3f00 {
		t.Error("Register is not 0x3f00")
	}

	ppu.Memory.Store(0x3f00, 0xff)
	ppu.Memory.Store(0x3f01, 0xff)
	ppu.Memory.Store(0x3f02, 0xff)

	if ppu.Fetch(0x2007) != 0xff {
		t.Error("Memory is not 0xff")
	}

	if ppu.Registers.Address != 0x3f01 {
		t.Error("Register is not 0x3f01")
	}
}

func TestDataIncrement32(t *testing.T) {
	divisor := uint64(4)
	clock := m65go2.NewClock(1 * time.Nanosecond)
	ppu := NewRP2C02(clock, divisor, Horizontal)

	ppu.Store(0x2000, 0x04)

	ppu.Registers.Address = 0x0000
	ppu.Fetch(0x2002)

	ppu.Store(0x2006, 0x01)
	ppu.Store(0x2006, 0x00)

	if ppu.Registers.Address != 0x0100 {
		t.Error("Register is not 0x0100")
	}

	ppu.Store(0x2007, 0xff)
	ppu.Store(0x2007, 0xff)
	ppu.Store(0x2007, 0xff)

	if ppu.Memory.Fetch(0x0100) != 0xff {
		t.Error("Memory is not 0xff")
	}

	if ppu.Memory.Fetch(0x0120) != 0xff {
		t.Error("Memory is not 0xff")
	}

	if ppu.Memory.Fetch(0x0140) != 0xff {
		t.Error("Memory is not 0xff")
	}
}
