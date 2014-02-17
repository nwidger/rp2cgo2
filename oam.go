package rp2cgo2

import (
	_ "fmt"
	"github.com/nwidger/m65go2"
)

type OAM struct {
	*m65go2.BasicMemory
	address    uint16
	latch      uint8
	buffer     *m65go2.BasicMemory
	index      uint16
	readCycle  func(oam *OAM, scanline uint16, cycle uint64, size uint16)
	writeCycle func(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool)
}

func NewOAM() *OAM {
	return &OAM{
		BasicMemory: m65go2.NewBasicMemory(256),
		buffer:      m65go2.NewBasicMemory(32),
		readCycle:   fetchAddress,
		writeCycle:  failCopyYPosition,
	}
}

func (oam *OAM) SpriteEvaluation(scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	if (scanline >= 0 && scanline <= 239) && (cycle >= 1 && cycle <= 256) {
		switch cycle {
		case 1:
			oam.address = 0
			oam.latch = 0xff
			oam.index = 0

			oam.DisableReads()
			oam.writeCycle = clearBuffer
		case 65:
			oam.address = 0
			oam.latch = 0xff
			oam.index = 0

			oam.EnableReads()
			oam.writeCycle = copyYPosition
		}

		switch cycle & 0x1 {
		case 1: // odd cycle
			if oam.readCycle != nil {
				oam.readCycle(oam, scanline, cycle, size)
			}
		case 0: // even cycle
			if oam.writeCycle != nil {
				spriteOverflow = oam.writeCycle(oam, scanline, cycle, size)
			}
		}
	}

	return
}

func fetchAddress(oam *OAM, scanline uint16, cycle uint64, size uint16) {
	if oam.address < 0x0100 {
		oam.latch = oam.Fetch(oam.address)
	}
}

func clearBuffer(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.buffer.Store(oam.address, oam.latch)
	oam.address++

	return
}

func copyYPosition(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	if scanline-uint16(uint32(oam.latch)) < size {
		oam.buffer.Store(oam.index+0, oam.latch)
		oam.writeCycle = copyIndex
		oam.address++
	} else {
		oam.address += 4

		if oam.address == 0x0100 {
			oam.writeCycle = failCopyYPosition
		}
	}

	return
}

func copyIndex(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.buffer.Store(oam.index+1, oam.latch)
	oam.writeCycle = copyAttributes
	oam.address++
	return
}

func copyAttributes(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.buffer.Store(oam.index+2, oam.latch)
	oam.writeCycle = copyXPosition
	oam.address++
	return
}

func copyXPosition(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.buffer.Store(oam.index+3, oam.latch)

	oam.index += 4
	oam.address++

	switch {
	case oam.address == 0x0100:
		oam.writeCycle = failCopyYPosition
	case oam.index < 32:
		oam.writeCycle = copyYPosition
	default:
		oam.buffer.DisableWrites()
		oam.address &= 0x00fc
		oam.writeCycle = evaluateYPosition
	}

	return
}

func evaluateYPosition(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	if scanline-uint16(uint32(oam.latch)) < size {
		spriteOverflow = true
		oam.address = (oam.address + 1) & 0x00ff
		oam.writeCycle = evaluateIndex
	} else {
		oam.address = ((oam.address + 4) & 0x00fc) + ((oam.address + 1) & 0x0003)

		if oam.address <= 0x0005 {
			oam.address &= 0x00fc
			oam.writeCycle = failCopyYPosition
		}
	}

	return
}

func evaluateIndex(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.address = (oam.address + 1) & 0x00ff
	oam.writeCycle = evaluateAttributes
	return
}

func evaluateAttributes(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.address = (oam.address + 1) & 0x00ff
	oam.writeCycle = evaluateXPosition
	return
}

func evaluateXPosition(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.address = (oam.address + 1) & 0x00ff

	if (oam.address & 0x0003) == 0x0003 {
		oam.address++
	}

	oam.address &= 0x00fc
	oam.writeCycle = failCopyYPosition

	return
}

func failCopyYPosition(oam *OAM, scanline uint16, cycle uint64, size uint16) (spriteOverflow bool) {
	oam.address = (oam.address + 4) & 0x00ff

	return
}
