package ioreg

import (
	"github.com/Div9851/gba-go/internal/dma"
	"github.com/Div9851/gba-go/internal/irq"
	"github.com/Div9851/gba-go/internal/ppu"
)

type IOReg struct {
	buffer       [0x400]byte
	changed      [0x400]bool
	IRQ          *irq.IRQ
	PPU          *ppu.PPU
	DMA          [4]*dma.Channel
	shouldCommit bool
}

func NewIOReg(irq *irq.IRQ, ppu *ppu.PPU, dma [4]*dma.Channel) *IOReg {
	return &IOReg{
		IRQ: irq,
		PPU: ppu,
		DMA: dma,
	}
}

func (r *IOReg) Read8(addr uint32) byte {
	switch {
	case addr < 0x2: // DISPCNT
		b := addr * 8
		return byte((r.PPU.DISPCNT >> b) & 0xFF)
	case 0x4 <= addr && addr < 0x6: // DISPSTAT
		b := (addr - 0x4) * 8
		return byte((r.PPU.DISPSTAT >> b) & 0xFF)
	case 0x6 <= addr && addr < 0x8: // VCOUNT
		b := (addr - 0x6) * 8
		return byte((r.PPU.VCOUNT >> b) & 0xFF)
	case 0xBA <= addr && addr < 0xBC: // DMA0CNT_H
		b := (addr - 0xBA) * 8
		return byte((r.DMA[0].CNT_H >> b) & 0xFF)
	case 0xC6 <= addr && addr < 0xC8: // DMA1CNT_H
		b := (addr - 0xC6) * 8
		return byte((r.DMA[1].CNT_H >> b) & 0xFF)
	case 0xD2 <= addr && addr < 0xD4: // DMA2CNT_H
		b := (addr - 0xD2) * 8
		return byte((r.DMA[2].CNT_H >> b) & 0xFF)
	case 0xDE <= addr && addr < 0xE0: // DMA3CNT_H
		b := (addr - 0xDE) * 8
		return byte((r.DMA[3].CNT_H >> b) & 0xFF)
	case 0x200 <= addr && addr < 0x202: // IE
		b := (addr - 0x200) * 8
		return byte((r.IRQ.IE >> b) & 0xFF)
	case 0x202 <= addr && addr < 0x204: // IF
		b := (addr - 0x202) * 8
		return byte((r.IRQ.IF >> b) & 0xFF)
	case 0x208 <= addr && addr < 0x20C: // IME
		b := (addr - 0x208) * 8
		return byte((r.IRQ.IME >> b) & 0xFF)
	}
	// Unknown
	return 0xFF
}

func (r *IOReg) Write8(addr uint32, val byte) {
	r.buffer[addr] = val
	r.changed[addr] = true
	r.shouldCommit = true
}

func (r *IOReg) Write16(addr uint32, val uint16) {
	r.Write8(addr, byte(val&0xFF))
	r.Write8(addr+1, byte((val>>8)&0xFF))
}

func (r *IOReg) Write32(addr uint32, val uint32) {
	r.Write16(addr, uint16(val&0xFFFF))
	r.Write16(addr+2, uint16((val>>16)&0xFFFF))
}

func (r *IOReg) getMask8(addr uint32) byte {
	if r.changed[addr] {
		return 0xFF
	} else {
		return 0x0
	}
}

func (r *IOReg) getMask16(addr uint32) uint16 {
	low := uint16(r.getMask8(addr))
	high := uint16(r.getMask8(addr + 1))
	return high<<8 | low
}

func (r *IOReg) getMask32(addr uint32) uint32 {
	low := uint32(r.getMask16(addr))
	high := uint32(r.getMask16(addr + 2))
	return high<<16 | low
}

func (r *IOReg) readBuffer8(addr uint32) byte {
	return r.buffer[addr]
}

func (r *IOReg) readBuffer16(addr uint32) uint16 {
	low := uint16(r.readBuffer8(addr))
	high := uint16(r.readBuffer8(addr + 1))
	return high<<8 | low
}

func (r *IOReg) readBuffer32(addr uint32) uint32 {
	low := uint32(r.readBuffer16(addr))
	high := uint32(r.readBuffer16(addr + 2))
	return high<<16 | low
}

func (r *IOReg) Commit() {
	if !r.shouldCommit {
		return
	}
	if mask := r.getMask16(0x0); mask != 0 { // DISPCNT
		value := r.readBuffer16(0x0) & mask
		r.PPU.DISPCNT = (r.PPU.DISPCNT & ^mask) | value
	}
	if mask := r.getMask16(0x4); mask != 0 { // DISPSTAT
		mask &= 0xFFB8
		value := r.readBuffer16(0x4) & mask
		r.PPU.DISPSTAT = (r.PPU.DISPSTAT & ^mask) | value
	}
	if mask := r.getMask32(0xB0); mask != 0 { // DMA0SAD
		value := r.readBuffer32(0xB0) & mask
		r.DMA[0].SAD = (r.DMA[0].SAD & ^mask) | value
	}
	if mask := r.getMask32(0xBC); mask != 0 { // DMA1SAD
		value := r.readBuffer32(0xBC) & mask
		r.DMA[1].SAD = (r.DMA[1].SAD & ^mask) | value
	}
	if mask := r.getMask32(0xC8); mask != 0 { // DMA2SAD
		value := r.readBuffer32(0xC8) & mask
		r.DMA[2].SAD = (r.DMA[2].SAD & ^mask) | value
	}
	if mask := r.getMask32(0xD4); mask != 0 { // DMA3SAD
		value := r.readBuffer32(0xD4) & mask
		r.DMA[3].SAD = (r.DMA[3].SAD & ^mask) | value
	}
	if mask := r.getMask32(0xB4); mask != 0 { // DMA0DAD
		value := r.readBuffer32(0xB4) & mask
		r.DMA[0].DAD = (r.DMA[0].DAD & ^mask) | value
	}
	if mask := r.getMask32(0xC0); mask != 0 { // DMA1DAD
		value := r.readBuffer32(0xC0) & mask
		r.DMA[1].DAD = (r.DMA[1].DAD & ^mask) | value
	}
	if mask := r.getMask32(0xCC); mask != 0 { // DMA2DAD
		value := r.readBuffer32(0xCC) & mask
		r.DMA[2].DAD = (r.DMA[2].DAD & ^mask) | value
	}
	if mask := r.getMask32(0xD8); mask != 0 { // DMA3DAD
		value := r.readBuffer32(0xD8) & mask
		r.DMA[3].DAD = (r.DMA[3].DAD & ^mask) | value
	}
	if mask := r.getMask16(0xB8); mask != 0 { // DMA0CNT_L
		value := r.readBuffer16(0xB8) & mask
		r.DMA[0].CNT_L = (r.DMA[0].CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0xC4); mask != 0 { // DMA1CNT_L
		value := r.readBuffer16(0xC4) & mask
		r.DMA[1].CNT_L = (r.DMA[1].CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0xD0); mask != 0 { // DMA2CNT_L
		value := r.readBuffer16(0xD0) & mask
		r.DMA[2].CNT_L = (r.DMA[2].CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0xDC); mask != 0 { // DMA3CNT_L
		value := r.readBuffer16(0xDC) & mask
		r.DMA[3].CNT_L = (r.DMA[3].CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0xBA); mask != 0 { // DMA0CNT_H
		value := r.readBuffer16(0xBA) & mask
		r.DMA[0].SetCNT_H((r.DMA[0].CNT_H & ^mask) | value)
	}
	if mask := r.getMask16(0xC6); mask != 0 { // DMA1CNT_H
		value := r.readBuffer16(0xC6) & mask
		r.DMA[1].SetCNT_H((r.DMA[1].CNT_H & ^mask) | value)
	}
	if mask := r.getMask16(0xD2); mask != 0 { // DMA2CNT_H
		value := r.readBuffer16(0xD2) & mask
		r.DMA[2].SetCNT_H((r.DMA[2].CNT_H & ^mask) | value)
	}
	if mask := r.getMask16(0xDE); mask != 0 { // DMA3CNT_H
		value := r.readBuffer16(0xDE) & mask
		r.DMA[3].SetCNT_H((r.DMA[3].CNT_H & ^mask) | value)
	}
	if mask := r.getMask16(0x200); mask != 0 { // IE
		value := r.readBuffer16(0x200) & mask
		r.IRQ.IE = (r.IRQ.IE & ^mask) | value
	}
	if mask := r.getMask16(0x202); mask != 0 { // IF
		value := r.readBuffer16(0x202) & mask
		r.IRQ.IF &= ^value
	}
	if mask := r.getMask16(0x208); mask != 0 { // IME
		value := r.readBuffer16(0x208) & mask
		r.IRQ.IME = (r.IRQ.IME & ^mask) | value
	}
	r.shouldCommit = false
	for i := range 0x400 {
		r.changed[i] = false
	}
}
