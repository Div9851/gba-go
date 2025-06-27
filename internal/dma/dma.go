package dma

import (
	"github.com/Div9851/gba-go/internal/irq"
	"github.com/Div9851/gba-go/internal/memory"
)

type Channel struct {
	index  int
	SAD    uint32
	DAD    uint32
	CNT_L  uint16
	CNT_H  uint16
	memory memory.Memory
	irq    *irq.IRQ
}

func NewChannel(index int, memory memory.Memory, irq *irq.IRQ) *Channel {
	return &Channel{
		index:  index,
		memory: memory,
		irq:    irq,
	}
}

func (ch *Channel) SetCNT_H(value uint16) {
	ch.CNT_H = value
}

func (ch *Channel) Step() {
	if (ch.CNT_H & (1 << 15)) != 0 {
		ch.Trigger()
	}

}

func (ch *Channel) Trigger() {
	src := ch.SAD
	if ch.index == 0 {
		src &= 0x7FFFFFF
	} else {
		src &= 0xFFFFFFF
	}

	dst := ch.DAD
	if ch.index < 3 {
		dst &= 0x7FFFFFF
	} else {
		dst &= 0xFFFFFFF
	}

	var wordSize uint32
	if (ch.CNT_H & (1 << 10)) == 0 {
		wordSize = 2
	} else {
		wordSize = 4
	}

	wordCount := int(ch.CNT_L)
	if wordCount == 0 {
		if ch.index < 3 {
			wordCount = 0x4000
		} else {
			wordCount = 0x10000
		}
	}

	for i := 0; i < wordCount; i++ {
		if wordSize == 2 {
			value := ch.memory.Read16(src)
			ch.memory.Write16(dst, value)
		} else {
			value := ch.memory.Read32(src)
			ch.memory.Write32(dst, value)
		}
		// Source Addr Control
		switch (ch.CNT_H >> 7) & 0x3 {
		case 0: // Increment
			src += wordSize
		case 1: // Decrement
			src -= wordSize
		}
		// Dest Addr Control
		switch (ch.CNT_H >> 5) & 0x3 {
		case 0: // Increment
			dst += wordSize
		case 1: // Decrement
			dst -= wordSize
		}
	}

	if (ch.CNT_H & (1 << 9)) == 0 { // not repeat
		ch.CNT_H &= 0x7FFF
	}

	if (ch.CNT_H & (1 << 14)) != 0 {
		ch.irq.IF |= 1 << (8 + ch.index)
	}
}
