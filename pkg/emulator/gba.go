package emulator

import (
	"github.com/Div9851/gba-go/internal/bus"
	"github.com/Div9851/gba-go/internal/cpu"
	"github.com/Div9851/gba-go/internal/dma"
	"github.com/Div9851/gba-go/internal/gamepak"
	"github.com/Div9851/gba-go/internal/input"
	"github.com/Div9851/gba-go/internal/ioreg"
	"github.com/Div9851/gba-go/internal/irq"
	"github.com/Div9851/gba-go/internal/ppu"
	"github.com/Div9851/gba-go/internal/timer"
)

const (
	cyclesPerFrame = 280896
)

type GBA struct {
	CPU    *cpu.CPU
	Bus    *bus.Bus
	PPU    *ppu.PPU
	DMA    [4]*dma.Channel
	Input  *input.Input
	Timers [4]*timer.Timer

	running bool
}

func NewGBA() *GBA {
	bus := bus.NewBus()
	irq := irq.NewIRQ()
	cpu := cpu.NewCPU(bus, irq)
	dmaChannels := [4]*dma.Channel{}
	for i := 0; i < 4; i++ {
		dmaChannels[i] = dma.NewChannel(i, bus, irq)
	}
	ppu := ppu.NewPPU(irq, dmaChannels)
	input := input.NewInput(irq)
	timers := [4]*timer.Timer{}
	for i := 3; i >= 0; i-- {
		timers[i] = timer.NewTimer(i, irq)
		if i < 3 {
			timers[i].Next = timers[i+1]
		}
	}
	ioReg := ioreg.NewIOReg(irq, ppu, dmaChannels, input, timers)

	bus.Setup(ppu, ioReg)

	gba := &GBA{
		CPU:     cpu,
		Bus:     bus,
		PPU:     ppu,
		DMA:     dmaChannels,
		Input:   input,
		Timers:  timers,
		running: false,
	}

	return gba
}

func (gba *GBA) Start() {
	gba.CPU.ResetPipeline()
	gba.running = true
}

func (gba *GBA) Stop() {
	gba.running = false
}

func (gba *GBA) LoadBIOS(data []byte) {
	gba.Bus.LoadBIOS(data)
}

func (gba *GBA) LoadROM(data []byte) {
	gba.Bus.GamePak = gamepak.NewGamePak(data)
}

func (gba *GBA) Step() {
	activeDMAChannel := -1
	for ch := 0; ch < 4; ch++ {
		if gba.DMA[ch].Status == dma.Active {
			activeDMAChannel = ch
			break
		}
	}
	if activeDMAChannel == -1 {
		gba.CPU.Step()
		for ch := 0; ch < 4; ch++ {
			if gba.DMA[ch].Status == dma.Wait && gba.DMA[ch].Cond == dma.Immediate {
				gba.DMA[ch].Trigger()
			}
		}
		for ch := 0; ch < 4; ch++ {
			if gba.DMA[ch].Status == dma.Triggered {
				gba.DMA[ch].Status = dma.Active
				break
			}
		}
	} else {
		gba.DMA[activeDMAChannel].Step()
	}
	gba.PPU.Step()
	for i := 0; i < 4; i++ {
		gba.Timers[i].Step()
	}
}

func (gba *GBA) Update(keys []string) {
	if !gba.running {
		return
	}
	gba.Input.SetKeys(keys)
	for i := 0; i < cyclesPerFrame; i++ {
		gba.Step()
	}
}
