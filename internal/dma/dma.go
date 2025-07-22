package dma

import (
	"github.com/Div9851/gba-go/internal/irq"
	"github.com/Div9851/gba-go/internal/memory"
)

const (
	None = iota
	Immediate
	VBlank
	HBlank
	SoundFIFO
)

const (
	Idle = iota
	Wait
	Triggered
	Active
)

type Channel struct {
	index  int
	SAD    uint32
	DAD    uint32
	CNT_L  uint16
	CNT_H  uint16
	Memory memory.Memory
	IRQ    *irq.IRQ

	srcAddr    uint32
	dstAddr    uint32
	wordSize   uint32
	wordCount  int
	srcAddrCnt int
	dstAddrCnt int
	repeat     bool
	triggerIRQ bool
	cycles     int
	Cond       int
	Status     int
}

func NewChannel(index int, memory memory.Memory, irq *irq.IRQ) *Channel {
	return &Channel{
		index:  index,
		Memory: memory,
		IRQ:    irq,
	}
}

func (ch *Channel) SetCNT_H(value uint16) {
	oldValue := ch.CNT_H
	ch.CNT_H = value
	if (oldValue&(1<<15)) == 0 && (value&(1<<15)) != 0 {
		ch.Load()
		ch.cycles = 0
		ch.Status = Wait
	} else if (oldValue&(1<<15)) != 0 && (value&(1<<15)) == 0 {
		ch.cycles = 0
		ch.Status = Idle
	}
}

func (ch *Channel) Step() {
	ch.cycles++

	if ch.cycles >= 2*ch.wordCount+2 {
		for i := 0; i < ch.wordCount; i++ {
			if ch.wordSize == 2 {
				value := ch.Memory.Read16(ch.srcAddr)
				ch.Memory.Write16(ch.dstAddr, value)
			} else {
				value := ch.Memory.Read32(ch.srcAddr)
				ch.Memory.Write32(ch.dstAddr, value)
			}
			switch ch.srcAddrCnt {
			case 0: // Increment
				ch.srcAddr += ch.wordSize
			case 1: // Decrement
				ch.srcAddr -= ch.wordSize
			}
			switch ch.dstAddrCnt {
			case 0:
				ch.dstAddr += ch.wordSize
			case 1: // Decrement
				ch.dstAddr -= ch.wordSize
			case 3: // Increment + Reload
				ch.dstAddr += ch.wordSize
			}
		}

		if ch.triggerIRQ {
			ch.IRQ.IF |= 1 << (8 + ch.index)
		}

		if !ch.repeat { // not repeat
			ch.CNT_H &= 0x7FFF
			ch.cycles = 0
			ch.Status = Idle
		} else {
			ch.LoadWordCount()
			if ch.dstAddrCnt == 3 {
				ch.LoadDAD()
			}
			ch.cycles = 0
			ch.Status = Wait
		}
	}
}

func (ch *Channel) LoadSAD() {
	ch.srcAddr = ch.SAD
	if ch.index == 0 {
		ch.srcAddr &= 0x7FFFFFF
	} else {
		ch.srcAddr &= 0xFFFFFFF
	}
	if ch.wordSize == 2 {
		ch.srcAddr &= 0xFFFFFFFE
	} else {
		ch.srcAddr &= 0xFFFFFFFC
	}
}

func (ch *Channel) LoadDAD() {
	ch.dstAddr = ch.DAD
	if ch.index < 3 {
		ch.dstAddr &= 0x7FFFFFF
	} else {
		ch.dstAddr &= 0xFFFFFFF
	}
	if ch.wordSize == 2 {
		ch.dstAddr &= 0xFFFFFFFE
	} else {
		ch.dstAddr &= 0xFFFFFFFC
	}
}

func (ch *Channel) LoadWordCount() {
	if ch.Cond == SoundFIFO {
		ch.wordCount = 4
		return
	}
	ch.wordCount = int(ch.CNT_L)
	if ch.wordCount == 0 {
		if ch.index < 3 {
			ch.wordCount = 0x4000
		} else {
			ch.wordCount = 0x10000
		}
	}
}

func (ch *Channel) Load() {
	switch (ch.CNT_H >> 12) & 0x3 {
	case 0x0:
		ch.Cond = Immediate
	case 0x1:
		ch.Cond = VBlank
	case 0x2:
		ch.Cond = HBlank
	case 0x3:
		if 1 <= ch.index && ch.index <= 2 {
			ch.Cond = SoundFIFO
		}
	}

	if ch.Cond == SoundFIFO || (ch.CNT_H&(1<<10)) != 0 {
		ch.wordSize = 4
	} else {
		ch.wordSize = 2
	}

	ch.LoadSAD()
	ch.LoadDAD()

	ch.LoadWordCount()

	ch.srcAddrCnt = int((ch.CNT_H >> 7) & 0x3)
	if ch.Cond == SoundFIFO {
		ch.dstAddrCnt = 2 // Fixed
	} else {
		ch.dstAddrCnt = int((ch.CNT_H >> 5) & 0x3)
	}
	ch.repeat = (ch.CNT_H & (1 << 9)) != 0
	ch.triggerIRQ = (ch.CNT_H & (1 << 14)) != 0
}

func (ch *Channel) Trigger() {
	ch.Status = Triggered
}
