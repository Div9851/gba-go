package emulator

import (
	"github.com/Div9851/gba-go/internal/bus"
	"github.com/Div9851/gba-go/internal/cpu"
	"github.com/Div9851/gba-go/internal/dma"
	"github.com/Div9851/gba-go/internal/gamepak"
	"github.com/Div9851/gba-go/internal/ioreg"
	"github.com/Div9851/gba-go/internal/irq"
	"github.com/Div9851/gba-go/internal/ppu"
)

const (
	cyclesPerFrame = 280896
)

type GBA struct {
	CPU *cpu.CPU
	Bus *bus.Bus
	PPU *ppu.PPU
	DMA [4]*dma.Channel

	running bool
}

func NewGBA() *GBA {
	bus := bus.NewBus()
	irq := irq.NewIRQ()
	cpu := cpu.NewCPU(bus, irq)
	ppu := ppu.NewPPU(irq)
	dmaChannels := [4]*dma.Channel{}
	for i := 0; i < 4; i++ {
		dmaChannels[i] = dma.NewChannel(i, bus, irq)
	}
	ioReg := ioreg.NewIOReg(irq, ppu, dmaChannels)
	gamepak := gamepak.NewGamePak()

	bus.Setup(gamepak, ppu, ioReg)

	gba := &GBA{
		CPU:     cpu,
		Bus:     bus,
		PPU:     ppu,
		DMA:     dmaChannels,
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
	gba.Bus.LoadROM(data)
}

func (gba *GBA) Step() {
	gba.CPU.Step()
	gba.PPU.Step()
	for i := 0; i < 4; i++ {
		gba.DMA[i].Step()
	}
}

func (gba *GBA) Update() {
	if !gba.running {
		return
	}
	for i := 0; i < cyclesPerFrame; i++ {
		gba.Step()
	}
}
