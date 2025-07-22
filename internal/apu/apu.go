package apu

import (
	"github.com/Div9851/gba-go/internal/dma"
)

const (
	systemClock = 16 * 1024 * 1024
)

var waveDuty = [4][8]bool{
	{false, true, false, false, false, false, false, false}, // 12.5%
	{false, true, true, false, false, false, false, false},  // 25%
	{false, true, true, true, true, false, false, false},    // 50%
	{true, false, false, true, true, true, true, true},      // 75%
}

func CalculateDutyPeriod(frequency int) int {
	return 16 * (2048 - frequency)
}

type Channel1 struct {
	CNT_L uint16
	CNT_H uint16
	CNT_X uint16

	dutyCounter     int
	sweepCounter    int
	envelopeCounter int
	lengthCounter   int

	sweepPeriod    int
	envelopePeriod int
	lengthPeriod   int

	dutyStep  int
	frequency int
	volume    int

	enabled bool
}

func (ch *Channel1) Start() {
	ch.dutyStep = 0
	ch.frequency = int(ch.CNT_X & 0x7FF)
	ch.volume = int((ch.CNT_H >> 12) & 0xF)

	ch.sweepPeriod = systemClock / 128 * int((ch.CNT_L>>4)&0x7)
	ch.envelopePeriod = systemClock / 64 * int((ch.CNT_H>>8)&0x7)
	ch.lengthPeriod = systemClock / 256 * (64 - int(ch.CNT_H&0x3F))

	ch.dutyCounter = CalculateDutyPeriod(ch.frequency)
	ch.sweepCounter = ch.sweepPeriod
	ch.envelopeCounter = ch.envelopePeriod
	ch.lengthCounter = ch.lengthPeriod

	ch.enabled = true
}

func (ch *Channel1) Step() {
	ch.dutyCounter--
	if ch.dutyCounter <= 0 {
		ch.dutyCounter = CalculateDutyPeriod(ch.frequency)
		ch.dutyStep = (ch.dutyStep + 1) & 7
	}
	if ch.sweepPeriod > 0 {
		ch.sweepCounter--
		if ch.sweepCounter <= 0 {
			ch.sweepCounter = ch.sweepPeriod
			dx := ch.frequency >> int(ch.CNT_L&0x7)
			if (ch.CNT_L & (1 << 3)) == 0 {
				ch.frequency += dx
			} else {
				ch.frequency -= dx
			}
			if ch.frequency < 0 || ch.frequency > 2047 {
				ch.enabled = false
				return
			}
			ch.CNT_X = (ch.CNT_X & 0xF800) | uint16(ch.frequency)
		}
	}
	if ch.envelopePeriod > 0 {
		ch.envelopeCounter--
		if ch.envelopeCounter <= 0 {
			ch.envelopeCounter = ch.envelopePeriod
			if (ch.CNT_H & (1 << 11)) != 0 {
				ch.volume = min(ch.volume+1, 15)
			} else {
				ch.volume = max(ch.volume-1, 0)
			}
		}
	}
	if (ch.CNT_X & (1 << 14)) != 0 {
		ch.lengthCounter--
		if ch.lengthCounter <= 0 {
			ch.enabled = false
		}
	}
}

func (ch *Channel1) Output() int {
	pattern := int((ch.CNT_H >> 6) & 0x3)
	if ch.enabled && waveDuty[pattern][ch.dutyStep] {
		return ch.volume
	}
	return 0
}

type Channel2 struct {
	CNT_L uint16
	CNT_H uint16

	dutyCounter     int
	envelopeCounter int
	lengthCounter   int

	envelopePeriod int
	lengthPeriod   int

	dutyStep  int
	frequency int
	volume    int

	enabled bool
}

func (ch *Channel2) Start() {
	ch.dutyStep = 0
	ch.frequency = int(ch.CNT_H & 0x7FF)
	ch.volume = int((ch.CNT_L >> 12) & 0xF)

	ch.envelopePeriod = systemClock / 64 * int((ch.CNT_L>>8)&0x7)
	ch.lengthPeriod = systemClock / 256 * (64 - int(ch.CNT_L&0x3F))

	ch.dutyCounter = CalculateDutyPeriod(ch.frequency)
	ch.envelopeCounter = ch.envelopePeriod
	ch.lengthCounter = ch.lengthPeriod

	ch.enabled = true
}

func (ch *Channel2) Step() {
	ch.dutyCounter--
	if ch.dutyCounter <= 0 {
		ch.dutyCounter = CalculateDutyPeriod(ch.frequency)
		ch.dutyStep = (ch.dutyStep + 1) & 7
	}
	if ch.envelopePeriod > 0 {
		ch.envelopeCounter--
		if ch.envelopeCounter <= 0 {
			ch.envelopeCounter = ch.envelopePeriod
			if (ch.CNT_L & (1 << 11)) != 0 {
				ch.volume = min(ch.volume+1, 15)
			} else {
				ch.volume = max(ch.volume-1, 0)
			}
		}
	}
	if (ch.CNT_H & (1 << 14)) != 0 {
		ch.lengthCounter--
		if ch.lengthCounter <= 0 {
			ch.enabled = false
		}
	}
}

func (ch *Channel2) Output() int {
	pattern := int((ch.CNT_L >> 6) & 0x3)
	if ch.enabled && waveDuty[pattern][ch.dutyStep] {
		return ch.volume
	}
	return 0
}

type Channel3 struct {
	CNT_L uint16
	CNT_H uint16
	CNT_X uint16
	RAM   [32]byte

	stepCounter   int
	lengthCounter int

	stepPeriod   int
	lengthPeriod int

	waveIndex int

	enabled bool
}

func (ch *Channel3) Start() {
	ch.waveIndex = 0

	ch.stepPeriod = 2 * (2048 - int(ch.CNT_X&0x7FF))
	ch.lengthPeriod = systemClock / 256 * (256 - int(ch.CNT_H&0xFF))

	ch.stepCounter = ch.stepPeriod
	ch.lengthCounter = ch.lengthPeriod

	ch.enabled = true
}

func (ch *Channel3) Step() {
	ch.stepCounter--
	if ch.stepCounter <= 0 {
		ch.stepCounter = ch.stepPeriod
		ch.waveIndex = (ch.waveIndex + 1) & 0x3F
	}
	if (ch.CNT_X & (1 << 14)) != 0 {
		ch.lengthCounter--
		if ch.lengthCounter <= 0 {
			ch.enabled = false
		}
	}
}

func (ch *Channel3) Output() int {
	if ch.enabled && (ch.CNT_L&(1<<7)) != 0 {
		var volume int
		sampleIndex := ch.waveIndex / 2
		if (ch.waveIndex & 1) == 0 {
			volume = int((ch.RAM[sampleIndex] >> 4) & 0xF)
		} else {
			volume = int(ch.RAM[sampleIndex] & 0xF)
		}
		if (ch.CNT_H & (1 << 15)) != 0 {
			volume = volume * 3 / 4
		} else {
			switch (ch.CNT_H >> 13) & 0x3 {
			case 0:
				volume = 0
			case 1:
			case 2:
				volume /= 2
			case 3:
				volume /= 4
			}
		}
		return volume
	}
	return 0
}

type Channel4 struct {
	CNT_L uint16
	CNT_H uint16

	stepCounter     int
	envelopeCounter int
	lengthCounter   int

	stepPeriod     int
	envelopePeriod int
	lengthPeriod   int

	volume int
	state  int

	enabled bool
}

func (ch *Channel4) Start() {
	ch.volume = int((ch.CNT_L >> 12) & 0xF)
	if (ch.CNT_H & (1 << 3)) != 0 { // 7 bits
		ch.state = 0x40
	} else { // 15 bits
		ch.state = 0x4000
	}

	r := int(ch.CNT_H & 7)
	s := int((ch.CNT_H >> 4) & 7)

	ch.stepPeriod = 32
	if r == 0 {
		ch.stepPeriod /= 2
	} else {
		ch.stepPeriod *= r
	}
	ch.stepPeriod <<= s + 1
	ch.envelopePeriod = systemClock / 64 * int((ch.CNT_L>>8)&0x7)
	ch.lengthPeriod = systemClock / 256 * (64 - int(ch.CNT_L&0x3F))

	ch.stepCounter = ch.stepPeriod
	ch.envelopeCounter = ch.envelopePeriod
	ch.lengthCounter = ch.lengthPeriod

	ch.enabled = true
}

func (ch *Channel4) Step() {
	ch.stepCounter--
	if ch.stepCounter <= 0 {
		ch.stepCounter = ch.stepPeriod
		carry := (ch.state & 1) != 0
		ch.state >>= 1
		if carry {
			if (ch.CNT_H & (1 << 3)) != 0 { // 7 bits
				ch.state ^= 0x60
			} else { // 15 bits
				ch.state ^= 0x6000
			}
		}
	}
	if ch.envelopePeriod > 0 {
		ch.envelopeCounter--
		if ch.envelopeCounter <= 0 {
			ch.envelopeCounter = ch.envelopePeriod
			if (ch.CNT_L & (1 << 11)) != 0 {
				ch.volume = min(ch.volume+1, 15)
			} else {
				ch.volume = max(ch.volume-1, 0)
			}
		}
	}
	if (ch.CNT_H & (1 << 14)) != 0 {
		ch.lengthCounter--
		if ch.lengthCounter <= 0 {
			ch.enabled = false
		}
	}
}

func (ch *Channel4) Output() int {
	if ch.enabled && (ch.state&1) != 0 {
		return ch.volume
	}
	return 0
}

type APU struct {
	Channel1 *Channel1
	Channel2 *Channel2
	Channel3 *Channel3
	Channel4 *Channel4

	SOUNDCNT_L uint16
	SOUNDCNT_H uint16

	dmaSound   [2]int8
	FIFO       [2][]byte
	DMA        [4]*dma.Channel
	cycles     uint64
	StreamerCh chan float32
}

func NewAPU(dma [4]*dma.Channel) *APU {
	return &APU{
		Channel1: &Channel1{},
		Channel2: &Channel2{},
		Channel3: &Channel3{},
		Channel4: &Channel4{},
		DMA:      dma,
	}
}

func (apu *APU) FIFOPush(index int, value byte) {
	apu.FIFO[index] = append(apu.FIFO[index], value)
}

func (apu *APU) FIFOPop(index int) {
	if len(apu.FIFO[index]) > 0 {
		apu.dmaSound[index] = int8(apu.FIFO[index][0])
		apu.FIFO[index] = apu.FIFO[index][1:]
	}
	if len(apu.FIFO[index]) <= 16 {
		apu.TriggerDMA(index)
	}
}

func (apu *APU) TriggerDMA(index int) {
	if apu.DMA[index+1].Status == dma.Wait && apu.DMA[index+1].Cond == dma.SoundFIFO {
		apu.DMA[index+1].Trigger()
	}
}

func (apu *APU) FIFOReset(index int) {
	apu.FIFO[index] = apu.FIFO[index][:0]
	apu.TriggerDMA(index)
}

func (apu *APU) TimerTick(index int) {
	if int((apu.SOUNDCNT_H>>10)&1) == index {
		apu.FIFOPop(0)
	}
	if int((apu.SOUNDCNT_H>>14)&1) == index {
		apu.FIFOPop(1)
	}
}

func (apu *APU) Step() {
	apu.cycles++
	apu.Channel1.Step()
	apu.Channel2.Step()
	apu.Channel3.Step()
	apu.Channel4.Step()
	// system clock 16*1024*1024 ≒ 16.78 MHz
	// sampling rate is 32.768 KHz (system clock / 512)
	if (apu.cycles & 0x1FF) == 0 {
		apu.SendSample()
	}
}

func (apu *APU) SendSample() {
	ch1 := float32(apu.Channel1.Output()) / 15
	ch2 := float32(apu.Channel2.Output()) / 15
	ch3 := float32(apu.Channel3.Output()) / 15
	ch4 := float32(apu.Channel4.Output()) / 15
	chA := float32(apu.dmaSound[0]) / 128
	chB := float32(apu.dmaSound[1]) / 128
	sample := min(max(ch1+ch2+ch3+ch4+chA+chB, -1), 1)
	select {
	case apu.StreamerCh <- sample:
	default:
	}
	select {
	case apu.StreamerCh <- sample:
	default:
	}
}
