package bus

import (
	"github.com/Div9851/gba-go/internal/gamepak"
	"github.com/Div9851/gba-go/internal/ioreg"
	"github.com/Div9851/gba-go/internal/ppu"
)

type Bus struct {
	BIOS    [16 * 1024]byte
	EWRAM   [256 * 1024]byte
	IWRAM   [32 * 1024]byte
	GamePak *gamepak.GamePak
	PPU     *ppu.PPU
	IOReg   *ioreg.IOReg
}

func NewBus() *Bus {
	return &Bus{}
}

func (bus *Bus) Setup(ppu *ppu.PPU, ioReg *ioreg.IOReg) {
	bus.PPU = ppu
	bus.IOReg = ioReg
}

func (bus *Bus) LoadBIOS(data []byte) {
	copy(bus.BIOS[:], data)
}

func (bus *Bus) write8(addr uint32, val byte) {
	if 0x2000000 <= addr && addr < 0x3000000 {
		bus.EWRAM[(addr-0x2000000)&0x3FFFF] = val
	} else if 0x3000000 <= addr && addr < 0x4000000 {
		bus.IWRAM[(addr-0x3000000)&0x7FFF] = val
	} else if 0x4000000 <= addr && addr < 0x40003FE {
		bus.IOReg.Write8(addr-0x4000000, val)
	} else if 0x5000000 <= addr && addr < 0x6000000 {
		bus.PPU.PRAM[(addr-0x5000000)&0x3FF] = val
	} else if 0x6000000 <= addr && addr < 0x7000000 {
		offset := (addr - 0x06000000) & 0x1FFFF
		if offset >= 0x18000 {
			offset -= 0x8000
		}
		bus.PPU.VRAM[offset] = val
	} else if 0x7000000 <= addr && addr < 0x8000000 {
		bus.PPU.OAM[(addr-0x7000000)&0x3FF] = val
	} else if 0xE000000 <= addr && addr < 0xE010000 {
		bus.GamePak.Backup.Write8(addr-0xE000000, val)
	}
}

func (bus *Bus) Write8(addr uint32, val byte) {
	if 0x6000000 <= addr && addr < 0x7000000 {
		offset := (addr - 0x6000000) & 0x1FFFF
		if offset >= 0x18000 {
			offset -= 0x8000
		}
		bitmapMode := (bus.PPU.DISPCNT & 7) >= 3
		if (!bitmapMode && offset < 0x10000) || (bitmapMode && offset < 0x14000) {
			addr &= 0xFFFFFFFE
			bus.write8(addr, val)
			bus.write8(addr+1, val)
		}
		return
	}
	if 0x5000000 <= addr && addr < 0x6000000 {
		addr &= 0xFFFFFFFE
		bus.write8(addr, val)
		bus.write8(addr+1, val)
	}
	if 0x7000000 <= addr && addr < 0x8000000 {
		return
	}
	bus.write8(addr, val)
	bus.IOReg.Commit()
}

func (bus *Bus) Read8(addr uint32) byte {
	if addr < 0x4000 {
		return bus.BIOS[addr]
	} else if 0x2000000 <= addr && addr < 0x3000000 {
		return bus.EWRAM[(addr-0x2000000)&0x3FFFF]
	} else if 0x3000000 <= addr && addr < 0x4000000 {
		return bus.IWRAM[(addr-0x3000000)&0x7FFF]
	} else if 0x4000000 <= addr && addr < 0x40003FE {
		return bus.IOReg.Read8(addr - 0x4000000)
	} else if 0x5000000 <= addr && addr < 0x6000000 {
		return bus.PPU.PRAM[(addr-0x5000000)&0x3FF]
	} else if 0x6000000 <= addr && addr < 0x7000000 {
		offset := (addr - 0x06000000) & 0x1FFFF
		if offset >= 0x18000 {
			offset -= 0x8000
		}
		return bus.PPU.VRAM[offset]
	} else if 0x7000000 <= addr && addr < 0x8000000 {
		return bus.PPU.OAM[(addr-0x7000000)&0x3FF]
	} else if 0x8000000 <= addr && addr < 0xE000000 {
		return bus.GamePak.ROM[(addr-0x8000000)&0x1FFFFFF]
	} else if 0xE000000 <= addr && addr < 0xE010000 {
		return bus.GamePak.Backup.Read8(addr - 0xE000000)
	}
	return 0
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
