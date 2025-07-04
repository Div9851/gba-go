package memory

import (
	"github.com/Div9851/gba-go/internal/gamepak"
	"github.com/Div9851/gba-go/internal/ioreg"
	"github.com/Div9851/gba-go/internal/ppu"
)

type Bus struct {
	ROM     [16 * 1024]byte
	EWRAM   [256 * 1024]byte
	IWRAM   [32 * 1024]byte
	GamePak *gamepak.GamePak
	PPU     *ppu.PPU
	IOReg   *ioreg.IOReg
}

func NewBus(gamepak *gamepak.GamePak, ppu *ppu.PPU, ioReg *ioreg.IOReg) *Bus {
	return &Bus{
		GamePak: gamepak,
		PPU:     ppu,
		IOReg:   ioReg,
	}
}

func (bus *Bus) write8(addr uint32, val byte) {
	if addr < 0x4000 {
		bus.ROM[addr] = val
	} else if 0x2000000 <= addr && addr < 0x2040000 {
		bus.EWRAM[addr-0x2000000] = val
	} else if 0x3000000 <= addr && addr < 0x3008000 {
		bus.IWRAM[addr-0x3000000] = val
	} else if 0x4000000 <= addr && addr < 0x40003FE {
		bus.IOReg.Write8(addr-0x4000000, val)
	} else if 0x5000000 <= addr && addr < 0x5000400 {
		bus.PPU.PRAM[addr-0x5000000] = val
	} else if 0x6000000 <= addr && addr < 0x6018000 {
		bus.PPU.VRAM[addr-0x6000000] = val
	} else if 0x7000000 <= addr && addr < 0x7000400 {
		bus.PPU.OAM[addr-0x7000000] = val
	} else if 0x8000000 <= addr && addr < 0xE000000 {
		bus.GamePak.ROM[addr-0x8000000] = val
	} else if 0xE000000 <= addr && addr < 0xE010000 {
		bus.GamePak.SRAM[addr-0xE000000] = val
	}
}

func (bus *Bus) Write8(addr uint32, val byte) {
	bus.write8(addr, val)
	bus.IOReg.Commit()
}

func (bus *Bus) Read8(addr uint32) byte {
	if addr < 0x4000 {
		return bus.ROM[addr]
	} else if 0x2000000 <= addr && addr < 0x2040000 {
		return bus.EWRAM[addr-0x2000000]
	} else if 0x3000000 <= addr && addr < 0x3008000 {
		return bus.IWRAM[addr-0x3000000]
	} else if 0x4000000 <= addr && addr < 0x40003FE {
		return bus.IOReg.Read8(addr - 0x4000000)
	} else if 0x5000000 <= addr && addr < 0x5000400 {
		return bus.PPU.PRAM[addr-0x5000000]
	} else if 0x6000000 <= addr && addr < 0x6018000 {
		return bus.PPU.VRAM[addr-0x6000000]
	} else if 0x7000000 <= addr && addr < 0x7000400 {
		return bus.PPU.OAM[addr-0x7000000]
	} else if 0x8000000 <= addr && addr < 0xE000000 {
		return bus.GamePak.ROM[addr-0x8000000]
	} else if 0xE000000 <= addr && addr < 0xE010000 {
		return bus.GamePak.SRAM[addr-0xE000000]
	}
	// Not Used
	return 0xFF
}

func (bus *Bus) Write16(addr uint32, val uint16) {
	bus.write8(addr, byte(val&0xFF))
	bus.write8(addr+1, byte((val>>8)&0xFF))
	bus.IOReg.Commit()
}

func (bus *Bus) Read16(addr uint32) uint16 {
	low := uint16(bus.Read8(addr))
	high := uint16(bus.Read8(addr + 1))
	return (high << 8) | low
}

func (bus *Bus) Write32(addr uint32, val uint32) {
	bus.write8(addr, byte(val&0xFF))
	bus.write8(addr+1, byte((val>>8)&0xFF))
	bus.write8(addr+2, byte((val>>16)&0xFF))
	bus.write8(addr+3, byte((val>>24)&0xFF))
	bus.IOReg.Commit()
}

func (bus *Bus) Read32(addr uint32) uint32 {
	low := uint32(bus.Read16(addr))
	high := uint32(bus.Read16(addr + 2))
	return (high << 16) | low
}
