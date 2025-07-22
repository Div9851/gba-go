package ioreg

import (
	"github.com/Div9851/gba-go/internal/apu"
	"github.com/Div9851/gba-go/internal/dma"
	"github.com/Div9851/gba-go/internal/input"
	"github.com/Div9851/gba-go/internal/irq"
	"github.com/Div9851/gba-go/internal/ppu"
	"github.com/Div9851/gba-go/internal/timer"
)

type IOReg struct {
	buffer       [0x400]byte
	changed      [0x400]bool
	IRQ          *irq.IRQ
	PPU          *ppu.PPU
	APU          *apu.APU
	DMA          [4]*dma.Channel
	Input        *input.Input
	Timers       [4]*timer.Timer
	shouldCommit bool
}

func NewIOReg(irq *irq.IRQ, ppu *ppu.PPU, apu *apu.APU, dma [4]*dma.Channel, input *input.Input, timers [4]*timer.Timer) *IOReg {
	return &IOReg{
		IRQ:    irq,
		PPU:    ppu,
		APU:    apu,
		DMA:    dma,
		Input:  input,
		Timers: timers,
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
	case 0x8 <= addr && addr < 0xA: // BG0CNT
		b := (addr - 0x8) * 8
		return byte((r.PPU.BGCNT[0] >> b) & 0xFF)
	case 0xA <= addr && addr < 0xC: // BG1CNT
		b := (addr - 0xA) * 8
		return byte((r.PPU.BGCNT[1] >> b) & 0xFF)
	case 0xC <= addr && addr < 0xE: // BG2CNT
		b := (addr - 0xC) * 8
		return byte((r.PPU.BGCNT[2] >> b) & 0xFF)
	case 0xE <= addr && addr < 0x10: // BG3CNT
		b := (addr - 0xE) * 8
		return byte((r.PPU.BGCNT[3] >> b) & 0xFF)
	case 0x60 <= addr && addr < 0x62: // SOUND1CNT_L
		b := (addr - 0x60) * 8
		return byte((r.APU.Channel1.CNT_L >> b) & 0xFF)
	case 0x62 <= addr && addr < 0x64: // SOUND1CNT_H
		b := (addr - 0x62) * 8
		return byte((r.APU.Channel1.CNT_H >> b) & 0xFF)
	case 0x64 <= addr && addr < 0x66: // SOUND1CNT_X
		b := (addr - 0x64) * 8
		return byte((r.APU.Channel1.CNT_X >> b) & 0xFF)
	case 0x68 <= addr && addr < 0x6A: // SOUND2CNT_L
		b := (addr - 0x68) * 8
		return byte((r.APU.Channel2.CNT_L >> b) & 0xFF)
	case 0x6C <= addr && addr < 0x6E: // SOUND2CNT_H
		b := (addr - 0x6C) * 8
		return byte((r.APU.Channel2.CNT_H >> b) & 0xFF)
	case 0x70 <= addr && addr < 0x72: // SOUND3CNT_L
		b := (addr - 0x70) * 8
		return byte((r.APU.Channel3.CNT_L >> b) & 0xFF)
	case 0x72 <= addr && addr < 0x74: // SOUND3CNT_H
		b := (addr - 0x72) * 8
		return byte((r.APU.Channel3.CNT_H >> b) & 0xFF)
	case 0x74 <= addr && addr < 0x76: // SOUND3CNT_X
		b := (addr - 0x74) * 8
		return byte((r.APU.Channel3.CNT_X >> b) & 0xFF)
	case 0x78 <= addr && addr < 0x7A: // SOUND4CNT_L
		b := (addr - 0x78) * 8
		return byte((r.APU.Channel4.CNT_L >> b) & 0xFF)
	case 0x7C <= addr && addr < 0x7E: // SOUND4CNT_H
		b := (addr - 0x7C) * 8
		return byte((r.APU.Channel4.CNT_H >> b) & 0xFF)
	case 0x90 <= addr && addr < 0xA0: // WAVE_RAM
		index := addr - 0x90
		if (r.APU.Channel3.CNT_L & (1 << 6)) == 0 {
			index += 16
		}
		return r.APU.Channel3.RAM[index]
	case 0x80 <= addr && addr < 0x82: // SOUNDCNT_L
		b := (addr - 0x80) * 8
		return byte((r.APU.SOUNDCNT_L >> b) & 0xFF)
	case 0x82 <= addr && addr < 0x84: // SOUNDCNT_H
		b := (addr - 0x82) * 8
		return byte((r.APU.SOUNDCNT_H >> b) & 0xFF)
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
	case 0x100 <= addr && addr < 0x102: // TM0CNT_L
		b := (addr - 0x100) * 8
		return byte((r.Timers[0].TMCNT_L >> b) & 0xFF)
	case 0x104 <= addr && addr < 0x106: // TM1CNT_L
		b := (addr - 0x104) * 8
		return byte((r.Timers[1].TMCNT_L >> b) & 0xFF)
	case 0x108 <= addr && addr < 0x10A: // TM2CNT_L
		b := (addr - 0x108) * 8
		return byte((r.Timers[2].TMCNT_L >> b) & 0xFF)
	case 0x10C <= addr && addr < 0x10E: // TM3CNT_L
		b := (addr - 0x10C) * 8
		return byte((r.Timers[3].TMCNT_L >> b) & 0xFF)
	case 0x102 <= addr && addr < 0x104: // TM0CNT_H
		b := (addr - 0x102) * 8
		return byte((r.Timers[0].TMCNT_H >> b) & 0xFF)
	case 0x106 <= addr && addr < 0x108: // TM1CNT_H
		b := (addr - 0x106) * 8
		return byte((r.Timers[1].TMCNT_H >> b) & 0xFF)
	case 0x10A <= addr && addr < 0x10C: // TM2CNT_H
		b := (addr - 0x10A) * 8
		return byte((r.Timers[2].TMCNT_H >> b) & 0xFF)
	case 0x10E <= addr && addr < 0x110: // TM3CNT_H
		b := (addr - 0x10E) * 8
		return byte((r.Timers[3].TMCNT_H >> b) & 0xFF)
	case 0x130 <= addr && addr < 0x132: // KEYINPUT
		b := (addr - 0x130) * 8
		return byte((r.Input.KEYINPUT >> b) & 0xFF)
	case 0x132 <= addr && addr < 0x134: // KEYCNT
		b := (addr - 0x132) * 8
		return byte((r.Input.KEYCNT >> b) & 0xFF)
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
	if mask := r.getMask16(0x8); mask != 0 { // BG0CNT
		value := r.readBuffer16(0x8) & mask
		r.PPU.BGCNT[0] = (r.PPU.BGCNT[0] & ^mask) | value
	}
	if mask := r.getMask16(0xA); mask != 0 { // BG1CNT
		value := r.readBuffer16(0xA) & mask
		r.PPU.BGCNT[1] = (r.PPU.BGCNT[1] & ^mask) | value
	}
	if mask := r.getMask16(0xC); mask != 0 { // BG2CNT
		value := r.readBuffer16(0xC) & mask
		r.PPU.BGCNT[2] = (r.PPU.BGCNT[2] & ^mask) | value
	}
	if mask := r.getMask16(0xE); mask != 0 { // BG3CNT
		value := r.readBuffer16(0xE) & mask
		r.PPU.BGCNT[3] = (r.PPU.BGCNT[3] & ^mask) | value
	}
	if mask := r.getMask16(0x10); mask != 0 { // BG0HOFS
		value := r.readBuffer16(0x10) & mask
		r.PPU.BGHOFS[0] = (r.PPU.BGHOFS[0] & ^mask) | value
	}
	if mask := r.getMask16(0x12); mask != 0 { // BG0VOFS
		value := r.readBuffer16(0x12) & mask
		r.PPU.BGVOFS[0] = (r.PPU.BGVOFS[0] & ^mask) | value
	}
	if mask := r.getMask16(0x14); mask != 0 { // BG1HOFS
		value := r.readBuffer16(0x14) & mask
		r.PPU.BGHOFS[1] = (r.PPU.BGHOFS[1] & ^mask) | value
	}
	if mask := r.getMask16(0x16); mask != 0 { // BG1VOFS
		value := r.readBuffer16(0x16) & mask
		r.PPU.BGVOFS[1] = (r.PPU.BGVOFS[1] & ^mask) | value
	}
	if mask := r.getMask16(0x18); mask != 0 { // BG2HOFS
		value := r.readBuffer16(0x18) & mask
		r.PPU.BGHOFS[2] = (r.PPU.BGHOFS[2] & ^mask) | value
	}
	if mask := r.getMask16(0x1A); mask != 0 { // BG2VOFS
		value := r.readBuffer16(0x1A) & mask
		r.PPU.BGVOFS[2] = (r.PPU.BGVOFS[2] & ^mask) | value
	}
	if mask := r.getMask16(0x1C); mask != 0 { // BG3HOFS
		value := r.readBuffer16(0x1C) & mask
		r.PPU.BGHOFS[3] = (r.PPU.BGHOFS[3] & ^mask) | value
	}
	if mask := r.getMask16(0x1E); mask != 0 { // BG3VOFS
		value := r.readBuffer16(0x1E) & mask
		r.PPU.BGVOFS[3] = (r.PPU.BGVOFS[3] & ^mask) | value
	}
	if mask := r.getMask16(0x28); mask != 0 { // BG2X_L
		value := r.readBuffer16(0x28) & mask
		r.PPU.BGX_L[2] = value
	}
	if mask := r.getMask16(0x2A); mask != 0 { // BG2X_H
		value := r.readBuffer16(0x2A) & mask
		r.PPU.BGX_H[2] = value
	}
	if mask := r.getMask16(0x2C); mask != 0 { // BG2Y_L
		value := r.readBuffer16(0x2C) & mask
		r.PPU.BGY_L[2] = value
	}
	if mask := r.getMask16(0x2E); mask != 0 { // BG2Y_H
		value := r.readBuffer16(0x2E) & mask
		r.PPU.BGY_H[2] = value
	}
	if mask := r.getMask16(0x20); mask != 0 { // BG2PA
		value := r.readBuffer16(0x20) & mask
		r.PPU.BG_PA[2] = value
	}
	if mask := r.getMask16(0x22); mask != 0 { // BG2PB
		value := r.readBuffer16(0x22) & mask
		r.PPU.BG_PB[2] = value
	}
	if mask := r.getMask16(0x24); mask != 0 { // BG2PC
		value := r.readBuffer16(0x24) & mask
		r.PPU.BG_PC[2] = value
	}
	if mask := r.getMask16(0x26); mask != 0 { // BG2PD
		value := r.readBuffer16(0x26) & mask
		r.PPU.BG_PD[2] = value
	}
	if mask := r.getMask16(0x60); mask != 0 { // SOUND1CNT_L
		value := r.readBuffer16(0x60) & mask
		r.APU.Channel1.CNT_L = (r.APU.Channel1.CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0x62); mask != 0 { // SOUND1CNT_H
		value := r.readBuffer16(0x62) & mask
		r.APU.Channel1.CNT_H = (r.APU.Channel1.CNT_H & ^mask) | value
	}
	if mask := r.getMask16(0x64); mask != 0 { // SOUND1CNT_X
		value := r.readBuffer16(0x64) & mask
		r.APU.Channel1.CNT_X = (r.APU.Channel1.CNT_X & ^mask) | value
		if (value & (1 << 15)) != 0 {
			r.APU.Channel1.Start()
		}
	}
	if mask := r.getMask16(0x68); mask != 0 { // SOUND2CNT_L
		value := r.readBuffer16(0x68) & mask
		r.APU.Channel2.CNT_L = (r.APU.Channel2.CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0x6C); mask != 0 { // SOUND2CNT_H
		value := r.readBuffer16(0x6C) & mask
		r.APU.Channel2.CNT_H = (r.APU.Channel2.CNT_H & ^mask) | value
		if (value & (1 << 15)) != 0 {
			r.APU.Channel2.Start()
		}
	}
	if mask := r.getMask16(0x70); mask != 0 { // SOUND3CNT_L
		value := r.readBuffer16(0x70) & mask
		r.APU.Channel3.CNT_L = (r.APU.Channel3.CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0x72); mask != 0 { // SOUND3CNT_H
		value := r.readBuffer16(0x72) & mask
		r.APU.Channel3.CNT_H = (r.APU.Channel3.CNT_H & ^mask) | value
	}
	if mask := r.getMask16(0x74); mask != 0 { // SOUND3CNT_X
		value := r.readBuffer16(0x74) & mask
		r.APU.Channel3.CNT_X = (r.APU.Channel3.CNT_X & ^mask) | value
		if (value & (1 << 15)) != 0 {
			r.APU.Channel3.Start()
		}
	}
	if mask := r.getMask16(0x78); mask != 0 { // SOUND4CNT_L
		value := r.readBuffer16(0x78) & mask
		r.APU.Channel4.CNT_L = (r.APU.Channel4.CNT_L & ^mask) | value
	}
	if mask := r.getMask16(0x7C); mask != 0 { // SOUND4CNT_H
		value := r.readBuffer16(0x7C) & mask
		r.APU.Channel4.CNT_H = (r.APU.Channel4.CNT_H & ^mask) | value
		if (value & (1 << 15)) != 0 {
			r.APU.Channel4.Start()
		}
	}
	for index := 0; index < 16; index++ {
		if r.changed[0x90+index] {
			if (r.APU.Channel3.CNT_L & (1 << 6)) == 0 {
				r.APU.Channel3.RAM[16+index] = r.buffer[0x90+index]
			} else {
				r.APU.Channel3.RAM[index] = r.buffer[0x90+index]
			}
		}
	}
	if mask := r.getMask16(0x80); mask != 0 { // SOUNDCNT_L
		value := r.readBuffer16(0x80) & mask
		r.APU.SOUNDCNT_L = (r.APU.SOUNDCNT_L & ^mask) | value
	}
	if mask := r.getMask16(0x82); mask != 0 { // SOUNDCNT_H
		value := r.readBuffer16(0x82) & mask
		r.APU.SOUNDCNT_H = (r.APU.SOUNDCNT_H & ^mask) | value
		if (value & (1 << 11)) != 0 {
			r.APU.FIFOReset(0)
		}
		if (value & (1 << 15)) != 0 {
			r.APU.FIFOReset(1)
		}
	}
	if mask := r.getMask16(0xA0); mask != 0 { // FIFO_A_L
		value := r.readBuffer16(0xA0) & mask
		r.APU.FIFOPush(0, byte(value&0xFF))
		r.APU.FIFOPush(0, byte((value>>8)&0xFF))
	}
	if mask := r.getMask16(0xA2); mask != 0 { // FIFO_A_H
		value := r.readBuffer16(0xA2) & mask
		r.APU.FIFOPush(0, byte(value&0xFF))
		r.APU.FIFOPush(0, byte((value>>8)&0xFF))
	}
	if mask := r.getMask16(0xA4); mask != 0 { // FIFO_B_L
		value := r.readBuffer16(0xA4) & mask
		r.APU.FIFOPush(1, byte(value&0xFF))
		r.APU.FIFOPush(1, byte((value>>8)&0xFF))
	}
	if mask := r.getMask16(0xA6); mask != 0 { // FIFO_B_H
		value := r.readBuffer16(0xA6) & mask
		r.APU.FIFOPush(1, byte(value&0xFF))
		r.APU.FIFOPush(1, byte((value>>8)&0xFF))
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
	if mask := r.getMask16(0x100); mask != 0 { // TM0CNT_L
		value := r.readBuffer16(0x100) & mask
		r.Timers[0].SetTMCNT_L(value)
	}
	if mask := r.getMask16(0x104); mask != 0 { // TM1CNT_L
		value := r.readBuffer16(0x104) & mask
		r.Timers[1].SetTMCNT_L(value)
	}
	if mask := r.getMask16(0x108); mask != 0 { // TM2CNT_L
		value := r.readBuffer16(0x108) & mask
		r.Timers[2].SetTMCNT_L(value)
	}
	if mask := r.getMask16(0x10C); mask != 0 { // TM3CNT_L
		value := r.readBuffer16(0x10C) & mask
		r.Timers[3].SetTMCNT_L(value)
	}
	if mask := r.getMask16(0x102); mask != 0 { // TM0CNT_H
		value := r.readBuffer16(0x102) & mask
		r.Timers[0].SetTMCNT_H(value)
	}
	if mask := r.getMask16(0x106); mask != 0 { // TM1CNT_H
		value := r.readBuffer16(0x106) & mask
		r.Timers[1].SetTMCNT_H(value)
	}
	if mask := r.getMask16(0x10A); mask != 0 { // TM2CNT_H
		value := r.readBuffer16(0x10A) & mask
		r.Timers[2].SetTMCNT_H(value)
	}
	if mask := r.getMask16(0x10E); mask != 0 { // TM3CNT_H
		value := r.readBuffer16(0x10E) & mask
		r.Timers[3].SetTMCNT_H(value)
	}
	if mask := r.getMask16(0x132); mask != 0 { // KEYCNT
		value := r.readBuffer16(0x132) & mask
		r.Input.KEYCNT = (r.Input.KEYCNT & ^mask) | value
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
