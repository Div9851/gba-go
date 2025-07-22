package timer

import (
	"github.com/Div9851/gba-go/internal/apu"
	"github.com/Div9851/gba-go/internal/irq"
)

type Timer struct {
	index   int
	TMCNT_L uint16
	TMCNT_H uint16
	reload  uint16
	IRQ     *irq.IRQ
	APU     *apu.APU
	cycles  int
	Next    *Timer
}

func NewTimer(index int, irq *irq.IRQ, apu *apu.APU) *Timer {
	return &Timer{
		IRQ: irq,
		APU: apu,
	}
}

func (tm *Timer) SetTMCNT_L(value uint16) {
	tm.reload = value
}

func (tm *Timer) SetTMCNT_H(value uint16) {
	oldValue := tm.TMCNT_H
	tm.TMCNT_H = value
	if (oldValue&(1<<7)) == 0 && (value&(1<<7)) != 0 { // start
		tm.cycles = 0
		tm.TMCNT_L = tm.reload
	}
}

func (tm *Timer) Step() {
	if (tm.TMCNT_H&(1<<7)) == 0 || (tm.TMCNT_H&(1<<2)) != 0 {
		return
	}

	var prescaler int
	switch tm.TMCNT_H & 0x3 {
	case 0x0:
		prescaler = 1
	case 0x1:
		prescaler = 64
	case 0x2:
		prescaler = 256
	case 0x3:
		prescaler = 1024
	}

	tm.cycles++
	if tm.cycles >= prescaler {
		tm.cycles = 0
		tm.Tick()
	}
}

func (tm *Timer) Tick() {
	if tm.TMCNT_L == 0xFFFF {
		if (tm.TMCNT_H & (1 << 6)) != 0 {
			tm.IRQ.IF |= 1 << (3 + tm.index)
		}
		if 0 <= tm.index && tm.index <= 1 {
			tm.APU.TimerTick(tm.index)
		}
		tm.TMCNT_L = tm.reload
		if tm.Next != nil && (tm.Next.TMCNT_H&(1<<2)) != 0 {
			tm.Next.Tick()
		}
	} else {
		tm.TMCNT_L++
	}
}
