package ioreg

import "github.com/Div9851/gba-go/internal/interrupt"

type IOReg struct {
	buffer       [0x400]byte
	changed      [0x400]bool
	shouldCommit bool
	interrupt    *interrupt.InterruptController
}

func NewIOReg(interrupt *interrupt.InterruptController) *IOReg {
	return &IOReg{
		interrupt: interrupt,
	}
}

func (r *IOReg) Read8(addr uint32) byte {
	if 0x200 <= addr && addr < 0x202 { // IE
		b := (addr - 0x200) * 8
		return byte((r.interrupt.IE >> b) & 0xFF)
	} else if 0x202 <= addr && addr < 0x204 { // IF
		b := (addr - 0x202) * 8
		return byte((r.interrupt.IF >> b) & 0xFF)
	} else if 0x208 <= addr && addr < 0x20C { // IME
		b := (addr - 0x208) * 8
		return byte((r.interrupt.IME >> b) & 0xFF)
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
	if mask := r.getMask16(0x200); mask != 0 { // IE
		r.interrupt.IE = r.readBuffer16(0x200) & mask
	}
	if mask := r.getMask16(0x202); mask != 0 { // IF
		value := r.readBuffer16(0x202) & mask
		r.interrupt.IF &= ^value
	}
	if mask := r.getMask32(0x208); mask != 0 { // IME
		r.interrupt.IME = r.readBuffer32(0x208) & mask
	}
	r.shouldCommit = false
	for i := range 0x400 {
		r.changed[i] = false
	}
}
