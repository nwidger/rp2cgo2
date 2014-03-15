package rp2cgo2

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"

	"github.com/nwidger/m65go2"
	"github.com/nwidger/rp2ago3"
)

type Mirroring uint8

const (
	Horizontal Mirroring = iota
	Vertical
	FourScreen
)

func (m Mirroring) String() string {
	switch m {
	case Horizontal:
		return "Horizontal"
	case Vertical:
		return "Vertical"
	case FourScreen:
		return "FourScreen"
	}

	return "Unknown"
}

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

type AddressFlag uint16

const (
	CoarseXScroll AddressFlag = 1 << iota
	_
	_
	_
	_
	CoarseYScroll
	_
	_
	_
	_
	NametableSelect
	_
	FineYScroll
	_
	_
	_
)

type SpriteFlag uint32

const (
	// byte 0
	YPosition SpriteFlag = 1 << iota
	_
	_
	_
	_
	_
	_
	_
	// byte 1
	TileBank
	TopTile
	_
	_
	_
	_
	_
	_
	// byte 2
	SpritePalette
	_
	_
	_
	_
	Priority
	FlipHorizontally
	FlipVertically
	// byte 3
	XPosition
	_
	_
	_
	_
	_
	_
	_
)

type Registers struct {
	Controller uint8
	Mask       uint8
	Status     uint8
	OAMAddress uint8
	Scroll     uint16
	Address    uint16
	Data       uint8
}

func (reg *Registers) Reset() {
	reg.Controller = 0x00
	reg.Mask = 0x00
	reg.Status = 0x00
	reg.OAMAddress = 0x00
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
	latch          bool
	latchAddress   uint16
	output         chan []uint8
	colors         []uint8
	Registers      Registers
	Memory         *rp2ago3.MappedMemory
	Interrupt      func(state bool)
	oam            *OAM
	frame          uint64
	scanline       uint16
	cycle          uint64
	patternAddress uint16
	attributeLatch uint8
	attributes     uint16
	tilesLatch     uint16
	tilesLow       uint16
	tilesHigh      uint16
	Cycles         chan uint16
	quota          uint16
}

func NewRP2C02(interrupt func(bool), mirroring Mirroring, output chan []uint8, cycles chan uint16) *RP2C02 {
	mem := rp2ago3.NewMappedMemory(m65go2.NewBasicMemory(m65go2.DEFAULT_MEMORY_SIZE))
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
		output:    output,
		Memory:    mem,
		Interrupt: interrupt,
		oam:       NewOAM(),
		Cycles:    cycles,
	}
}

func (ppu *RP2C02) Reset() {
	ppu.latch = false
	ppu.Registers.Reset()
	ppu.Memory.Reset()

	ppu.frame = 0
	ppu.cycle = 0
	ppu.scanline = POWERUP_SCANLINE
	ppu.quota = 0
}

func (ppu *RP2C02) controller(flag ControllerFlag) (value uint16) {
	byte := ppu.Registers.Controller
	bit := byte & uint8(flag)

	switch flag {
	case BaseNametableAddress:
		// 0x2000 | 0x2400 | 0x2800 | 0x2c00
		value = 0x2000 | (uint16(byte&0x03) << 10)
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

func (ppu *RP2C02) address(flag AddressFlag) (value uint16) {
	word := ppu.Registers.Address

	switch flag {
	case CoarseXScroll:
		value = word & 0x001f
	case CoarseYScroll:
		value = (word & 0x03e0) >> 5
	case NametableSelect:
		value = (word & 0x0c00) >> 10
	case FineYScroll:
		value = (word & 0x7000) >> 12
	}

	return
}

func (ppu *RP2C02) sprite(sprite uint32, flag SpriteFlag) (value uint8) {
	switch flag {
	case YPosition:
		value = uint8(sprite)
	case TileBank:
		value = uint8((sprite & 0x00000100) >> 8)
	case TopTile:
		value = uint8((sprite & 0x0000fe00) >> 9)
	case SpritePalette:
		value = uint8((sprite & 0x00030000) >> 16)
	case Priority:
		value = uint8((sprite & 0x00200000) >> 21)
	case FlipHorizontally:
		value = uint8((sprite & 0x00400000) >> 22)
	case FlipVertically:
		value = uint8((sprite & 0x00800000) >> 23)
	case XPosition:
		value = uint8(sprite >> 24)
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
		value = ppu.oam.Fetch(uint16(ppu.Registers.OAMAddress))
	// Data
	case 0x2007:
		value = ppu.Registers.Data

		vramAddress := ppu.Registers.Address & 0x3fff
		ppu.Registers.Data = ppu.Memory.Fetch(vramAddress)

		if vramAddress&0x3f00 == 0x3f00 {
			value = ppu.Registers.Data
		}

		ppu.incrementAddress()
	}

	return
}

func (ppu *RP2C02) Store(address uint16, value uint8) (oldValue uint8) {
	switch address {
	// Controller
	case 0x2000:
		// t: ...BA.. ........ = d: ......BA
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
		if !ppu.rendering() || ppu.scanline == 240 {
			oldValue = ppu.oam.Fetch(uint16(ppu.Registers.OAMAddress))
			ppu.oam.Store(uint16(ppu.Registers.OAMAddress), value)
			ppu.Registers.OAMAddress++
		}
	// Scroll
	case 0x2005:
		if !ppu.latch {
			// t: ....... ...HGFED = d: HGFED...
			// x:              CBA = d: .....CBA
			ppu.latchAddress = (ppu.latchAddress & 0x7fe0) | uint16(value>>3)
			ppu.Registers.Scroll = uint16(value & 0x07)
		} else {
			// t: CBA..HG FED..... = d: HGFEDCBA
			ppu.latchAddress = (ppu.latchAddress & 0x0c1f) | ((uint16(value)<<2 | uint16(value)<<12) & 0x73e0)
		}

		ppu.latch = !ppu.latch
	// Address
	case 0x2006:
		if !ppu.latch {
			// t: .FEDCBA ........ = d: ..FEDCBA
			// t: X...... ........ = 0
			ppu.latchAddress = (ppu.latchAddress & 0x00ff) | uint16(value&0x3f)<<8
		} else {
			// t: ....... HGFEDCBA = d: HGFEDCBA
			// v                   = t
			ppu.latchAddress = (ppu.latchAddress & 0x7f00) | uint16(value)
			ppu.Registers.Address = ppu.latchAddress
		}

		ppu.latch = !ppu.latch
	// Data
	case 0x2007:
		oldValue = ppu.Registers.Data
		ppu.Memory.Store(ppu.Registers.Address&0x3fff, value)
		ppu.incrementAddress()
	}

	return
}

func (ppu *RP2C02) transferX() {
	// v: ....F.. ...EDCBA = t: ....F.. ...EDCBA
	ppu.Registers.Address = (ppu.Registers.Address & 0x7be0) | (ppu.latchAddress & 0x041f)
}

func (ppu *RP2C02) transferY() {
	// v: IHGF.ED CBA..... = t: IHGF.ED CBA.....
	ppu.Registers.Address = (ppu.Registers.Address & 0x041f) | (ppu.latchAddress & 0x7be0)
}

func (ppu *RP2C02) incrementX() {
	// v: .yyy NN YYYYY XXXXX
	//     ||| || ||||| +++++-- coarse X scroll
	//     ||| || +++++-------- coarse Y scroll
	//     ||| ++-------------- nametable select
	//     +++----------------- fine Y scroll
	v := ppu.Registers.Address

	switch v & 0x001f {
	case 0x001f: // coarse X == 31
		v ^= 0x041f // coarse X = 0, switch horizontal nametable
	default:
		v++ // increment coarse X
	}

	ppu.Registers.Address = v
}

func (ppu *RP2C02) incrementY() {
	// v: .yyy NN YYYYY XXXXX
	//     ||| || ||||| +++++-- coarse X scroll
	//     ||| || +++++-------- coarse Y scroll
	//     ||| ++-------------- nametable select
	//     +++----------------- fine Y scroll
	v := ppu.Registers.Address

	if (v & 0x7000) != 0x7000 { // if fine Y < 7
		v += 0x1000 // increment fine Y
	} else {
		v &= 0x0fff

		switch v & 0x03e0 {
		case 0x03a0: // coarse Y = 29
			v ^= 0x0800 // switch vertical nametable
		case 0x03e0: // coarse Y = 31
			v &= 0x7c1f // coarse Y = 0, nametable not switched
		default:
			v += 0x0020 // increment coarse Y
		}
	}

	ppu.Registers.Address = v
}

func (ppu *RP2C02) incrementAddress() {
	if !ppu.rendering() || ppu.scanline == 240 {
		ppu.Registers.Address =
			(ppu.Registers.Address + ppu.controller(VRAMAddressIncrement)) & 0x7fff
	} else {
		if ppu.controller(VRAMAddressIncrement) == 32 {
			ppu.incrementY()
		} else {
			ppu.Registers.Address++
		}
	}
}

func (ppu *RP2C02) reloadBackgroundTiles() {
	ppu.tilesLow = (ppu.tilesLow & 0xff00) | (ppu.tilesLatch & 0x00ff)
	ppu.tilesHigh = (ppu.tilesHigh & 0xff00) | ((ppu.tilesLatch >> 8) & 0x00ff)
}

func (ppu *RP2C02) shiftBackgroundTiles() {
	ppu.tilesLow <<= 1
	ppu.tilesHigh <<= 1
	ppu.attributes = (ppu.attributes >> 2) | (uint16(ppu.attributeLatch) << 14)
}

func (ppu *RP2C02) rendering() bool {
	return ppu.mask(ShowBackground) || ppu.mask(ShowSprites)
}

func (ppu *RP2C02) fetchName(address uint16) (value uint8) {
	//               NNii iiii iiii
	// 0x2000 = 0010 0000 0000 0000
	// 0x2400 = 0010 0100 0000 0000
	// 0x2800 = 0010 1000 0000 0000
	// 0x2c00 = 0010 1100 0000 0000
	value = ppu.Memory.Fetch(0x2000 | address&0x0fff)

	return
}

func (ppu *RP2C02) fetchAttribute(address uint16) (value uint8) {
	// 0x23c0 = 0010 0011 1100 0000
	//               NN = 0x0c00
	//                      ii i = 0x0038
	//                          jjj = 0x0007
	value = ppu.Memory.Fetch(0x23c0 | (address & 0x0c00) | (address >> 4 & 0x0038) | (address >> 2 & 0x0007))

	return
}

var img *image.RGBA

func (ppu *RP2C02) renderVisibleScanline() {
	// fmt.Printf("======== cycle %v ========\n", cycle)
	switch ppu.cycle {
	// skipped on BG+odd
	case 0:
		if ppu.scanline == 0 && ppu.rendering() && ppu.frame&0x1 != 0 {
			ppu.quota++
		}

	// NT byte
	case 1:
		if ppu.scanline == 261 {
			ppu.Registers.Status &^= uint8(VBlankStarted | Sprite0Hit | SpriteOverflow)
		}

		fallthrough
	case 9:
		fallthrough
	case 17:
		fallthrough
	case 25:
		fallthrough
	case 33:
		fallthrough
	case 41:
		fallthrough
	case 49:
		fallthrough
	case 57:
		fallthrough
	case 65:
		fallthrough
	case 73:
		fallthrough
	case 81:
		fallthrough
	case 89:
		fallthrough
	case 97:
		fallthrough
	case 105:
		fallthrough
	case 113:
		fallthrough
	case 121:
		fallthrough
	case 129:
		fallthrough
	case 137:
		fallthrough
	case 145:
		fallthrough
	case 153:
		fallthrough
	case 161:
		fallthrough
	case 169:
		fallthrough
	case 177:
		fallthrough
	case 185:
		fallthrough
	case 193:
		fallthrough
	case 201:
		fallthrough
	case 209:
		fallthrough
	case 217:
		fallthrough
	case 225:
		fallthrough
	case 233:
		fallthrough
	case 241:
		fallthrough
	case 249:
		fallthrough
	case 321:
		fallthrough
	case 329:
		fallthrough
	case 337:
		// 000p NNNN NNNN vvvv
		if ppu.rendering() {
			ppu.reloadBackgroundTiles()
			ppu.patternAddress = ppu.controller(BackgroundPatternAddress) |
				uint16(ppu.fetchName(ppu.Registers.Address))<<4 |
				ppu.address(FineYScroll)
		}

	// AT byte
	case 3:
		fallthrough
	case 11:
		fallthrough
	case 19:
		fallthrough
	case 27:
		fallthrough
	case 35:
		fallthrough
	case 43:
		fallthrough
	case 51:
		fallthrough
	case 59:
		fallthrough
	case 67:
		fallthrough
	case 75:
		fallthrough
	case 83:
		fallthrough
	case 91:
		fallthrough
	case 99:
		fallthrough
	case 107:
		fallthrough
	case 115:
		fallthrough
	case 123:
		fallthrough
	case 131:
		fallthrough
	case 139:
		fallthrough
	case 147:
		fallthrough
	case 155:
		fallthrough
	case 163:
		fallthrough
	case 171:
		fallthrough
	case 179:
		fallthrough
	case 187:
		fallthrough
	case 195:
		fallthrough
	case 203:
		fallthrough
	case 211:
		fallthrough
	case 219:
		fallthrough
	case 227:
		fallthrough
	case 235:
		fallthrough
	case 243:
		fallthrough
	case 251:
		fallthrough
	case 323:
		fallthrough
	case 331:
		// combine 2nd X- and Y-bit of loopy_v to
		// determine which 2-bits of AT byte to use:
		//
		// value = (topleft << 0) | (topright << 2) | (bottomleft << 4) | (bottomright << 6)
		//
		// v: .yyy NNYY YYYX XXXX|
		//    .... .... .... ..X.|
		// v >> 4: .... .>>> >Y..|....
		//         .X. = 000 = 0
		//         Y..   010 = 2
		//               100 = 4
		//               110 = 6
		if ppu.rendering() {
			ppu.attributeLatch = (ppu.fetchAttribute(ppu.Registers.Address) >>
				((ppu.Registers.Address & 0x2) | (ppu.Registers.Address >> 4 & 0x4))) & 0x03
		}

	// Low BG tile byte (color bit 0)
	case 5:
		fallthrough
	case 13:
		fallthrough
	case 21:
		fallthrough
	case 29:
		fallthrough
	case 37:
		fallthrough
	case 45:
		fallthrough
	case 53:
		fallthrough
	case 61:
		fallthrough
	case 69:
		fallthrough
	case 77:
		fallthrough
	case 85:
		fallthrough
	case 93:
		fallthrough
	case 101:
		fallthrough
	case 109:
		fallthrough
	case 117:
		fallthrough
	case 125:
		fallthrough
	case 133:
		fallthrough
	case 141:
		fallthrough
	case 149:
		fallthrough
	case 157:
		fallthrough
	case 165:
		fallthrough
	case 173:
		fallthrough
	case 181:
		fallthrough
	case 189:
		fallthrough
	case 197:
		fallthrough
	case 205:
		fallthrough
	case 213:
		fallthrough
	case 221:
		fallthrough
	case 229:
		fallthrough
	case 237:
		fallthrough
	case 245:
		fallthrough
	case 253:
		fallthrough
	case 325:
		fallthrough
	case 333:
		if ppu.rendering() {
			// Fetch color bit 0 for next 8 dots
			ppu.tilesLatch = (ppu.tilesLatch & 0xff00) | uint16(ppu.Memory.Fetch(ppu.patternAddress))
		}

	// High BG tile byte (color bit 1)
	case 7:
		fallthrough
	case 15:
		fallthrough
	case 23:
		fallthrough
	case 31:
		fallthrough
	case 39:
		fallthrough
	case 47:
		fallthrough
	case 55:
		fallthrough
	case 63:
		fallthrough
	case 71:
		fallthrough
	case 79:
		fallthrough
	case 87:
		fallthrough
	case 95:
		fallthrough
	case 103:
		fallthrough
	case 111:
		fallthrough
	case 119:
		fallthrough
	case 127:
		fallthrough
	case 135:
		fallthrough
	case 143:
		fallthrough
	case 151:
		fallthrough
	case 159:
		fallthrough
	case 167:
		fallthrough
	case 175:
		fallthrough
	case 183:
		fallthrough
	case 191:
		fallthrough
	case 199:
		fallthrough
	case 207:
		fallthrough
	case 215:
		fallthrough
	case 223:
		fallthrough
	case 231:
		fallthrough
	case 239:
		fallthrough
	case 247:
		fallthrough
	case 255:
		fallthrough
	case 327:
		fallthrough
	case 335:
		if ppu.rendering() {
			// Fetch color bit 1 for next 8 dots
			ppu.tilesLatch = (ppu.tilesLatch & 0x00ff) | uint16(ppu.Memory.Fetch(ppu.patternAddress|0x0008))<<8
		}

	// inc hori(v)
	case 8:
		fallthrough
	case 16:
		fallthrough
	case 24:
		fallthrough
	case 32:
		fallthrough
	case 40:
		fallthrough
	case 48:
		fallthrough
	case 56:
		fallthrough
	case 64:
		fallthrough
	case 72:
		fallthrough
	case 80:
		fallthrough
	case 88:
		fallthrough
	case 96:
		fallthrough
	case 104:
		fallthrough
	case 112:
		fallthrough
	case 120:
		fallthrough
	case 128:
		fallthrough
	case 136:
		fallthrough
	case 144:
		fallthrough
	case 152:
		fallthrough
	case 160:
		fallthrough
	case 168:
		fallthrough
	case 176:
		fallthrough
	case 184:
		fallthrough
	case 192:
		fallthrough
	case 200:
		fallthrough
	case 208:
		fallthrough
	case 216:
		fallthrough
	case 224:
		fallthrough
	case 232:
		fallthrough
	case 240:
		fallthrough
	case 248:
		fallthrough
	case 328:
		fallthrough
	case 336:
		if ppu.rendering() {
			ppu.incrementX()
		}

	// inc vert(v)
	case 256:
		if ppu.rendering() {
			ppu.incrementY()
		}

	// hori(v) = hori(t)
	case 257:
		if ppu.rendering() {
			ppu.reloadBackgroundTiles()
			ppu.transferX()
		}

	// vert(v) = vert(t)
	case 280:
		fallthrough
	case 281:
		fallthrough
	case 282:
		fallthrough
	case 283:
		fallthrough
	case 284:
		fallthrough
	case 285:
		fallthrough
	case 286:
		fallthrough
	case 287:
		fallthrough
	case 288:
		fallthrough
	case 289:
		fallthrough
	case 290:
		fallthrough
	case 291:
		fallthrough
	case 292:
		fallthrough
	case 293:
		fallthrough
	case 294:
		fallthrough
	case 295:
		fallthrough
	case 296:
		fallthrough
	case 297:
		fallthrough
	case 298:
		fallthrough
	case 299:
		fallthrough
	case 300:
		fallthrough
	case 301:
		fallthrough
	case 302:
		fallthrough
	case 303:
		fallthrough
	case 304:
		if ppu.scanline == 261 && ppu.rendering() {
			ppu.transferY()
		}
	}

	if ppu.cycle >= 1 && ppu.cycle <= 256 {
		index := uint16(0)
		attribute := uint16(0)

		if ppu.mask(ShowBackground) && (ppu.mask(ShowBackgroundLeft) || ppu.cycle > 8) {
			scroll := 9 + ppu.Registers.Scroll
			index = (((ppu.tilesHigh >> scroll) & 0x0001) << 1) | ((ppu.tilesLow >> scroll) & 0x0001)

			if index != 0 {
				attribute = uint16((ppu.attributes)&0x0003) << 2
			}
		}

		color := ppu.Memory.Fetch(0x3f00 | attribute | index)

		if ppu.rendering() && ppu.scanline >= 0 && ppu.scanline <= 239 {
			ppu.colors = append(ppu.colors, color)
		}

		if ppu.oam.SpriteEvaluation(ppu.scanline, ppu.cycle, ppu.controller(SpriteSize)) {
			ppu.Registers.Status |= uint8(SpriteOverflow)
		}
	}

	if (ppu.cycle >= 2 && ppu.cycle <= 257) || (ppu.cycle >= 322 && ppu.cycle <= 337) {
		ppu.shiftBackgroundTiles()
	}

	return
}

func (ppu *RP2C02) Execute() {
	if ppu.quota == 0 {
		ppu.quota = <-ppu.Cycles
	}

	switch {
	// visible scanlines (0-239), post-render scanline (240), pre-render scanline (261)
	case ppu.scanline < 241 || ppu.scanline > 260:
		ppu.renderVisibleScanline()
	// vertical blanking scanlines (241-260)
	default:
		if ppu.scanline == 241 && ppu.cycle == 1 {
			ppu.Registers.Status |= uint8(VBlankStarted)

			if ppu.Registers.Status&uint8(VBlankStarted) != 0 &&
				ppu.Registers.Controller&uint8(NMIOnVBlank) != 0 {
				if ppu.Interrupt != nil {
					ppu.Interrupt(true)
				}
			}
		}
	}

	ppu.quota--
	if ppu.quota == 0 {
		ppu.Cycles <- 1
	}
}

func (ppu *RP2C02) dumpPatternTables() (left, right *image.RGBA) {
	left = image.NewRGBA(image.Rect(0, 0, 128, 128))
	right = image.NewRGBA(image.Rect(0, 0, 128, 128))

	colors := [4]color.RGBA{
		color.RGBA{0, 0, 0, 255},
		color.RGBA{203, 79, 15, 255},
		color.RGBA{255, 155, 59, 255},
		color.RGBA{255, 231, 163, 255},
	}

	x_base := 0
	y_base := 0

	ptimg := left

	for address := uint16(0x0000); address <= 0x1fff; address += 0x0010 {
		if address < 0x1000 {
			ptimg = left
		} else {
			ptimg = right
		}

		for row := uint16(0); row <= 7; row++ {
			low := ppu.Memory.Fetch(address + row)
			high := ppu.Memory.Fetch(address + row + 8)

			for i := int16(7); i >= 0; i-- {
				b := ((low >> uint16(i)) & 0x0001) | (((high >> uint16(i)) & 0x0001) << 1)
				ptimg.Set(x_base+(8-int(i+1)), y_base+int(row), colors[b])
			}
		}

		x_base += 8

		if x_base == 128 {
			x_base = 0
			y_base = (y_base + 8) % 128
		}
	}

	fo, _ := os.Create(fmt.Sprintf("left.jpg"))
	w := bufio.NewWriter(fo)
	jpeg.Encode(w, left, &jpeg.Options{Quality: 100})

	fo, _ = os.Create(fmt.Sprintf("right.jpg"))
	w = bufio.NewWriter(fo)
	jpeg.Encode(w, right, &jpeg.Options{Quality: 100})

	return
}

func (ppu *RP2C02) Run() {
	ppu.dumpPatternTables()
	img = image.NewRGBA(image.Rect(0, 0, 256, 240))

	for {
		// fmt.Printf("******** frame %v ********\n", ppu.frame)

		ppu.colors = []uint8{}

		for ; ppu.scanline < NUM_SCANLINES; ppu.scanline++ {
			if ppu.scanline == 8 {
				ppu.Registers.Status |= uint8(Sprite0Hit)
			}

			for ppu.cycle = 0; ppu.cycle < CYCLES_PER_SCANLINE; ppu.cycle++ {
				ppu.Execute()
			}
		}

		ppu.Registers.Status &^= uint8(Sprite0Hit)

		if ppu.rendering() {
			ppu.output <- ppu.colors
			<-ppu.output
		}

		ppu.scanline = 0
		ppu.frame++
	}
}
