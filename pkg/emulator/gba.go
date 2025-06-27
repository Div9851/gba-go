package emulator

import (
	"github.com/Div9851/gba-go/internal/cpu"
	"github.com/Div9851/gba-go/internal/gamepak"
	"github.com/Div9851/gba-go/internal/interrupt"
	"github.com/Div9851/gba-go/internal/ioreg"
	"github.com/Div9851/gba-go/internal/memory"
	"github.com/Div9851/gba-go/internal/ppu"
)

const (
	cyclesPerFrame = 280896
)

type GBA struct {
	CPU *cpu.CPU
	Bus *memory.Bus
	PPU *ppu.PPU

	running bool
}

func NewGBA() *GBA {
	gamepak := gamepak.NewGamePak()
	interrupt := interrupt.NewInterruptController()
	ppu := ppu.NewPPU()
	ioReg := ioreg.NewIOReg(interrupt)
	bus := memory.NewBus(gamepak, ppu, ioReg)
	cpu := cpu.NewCPU(bus, interrupt)

	gba := &GBA{
		CPU:     cpu,
		Bus:     bus,
		PPU:     ppu,
		running: false,
	}

	return gba
}

func (gba *GBA) Update() {
	if !gba.running {
		return
	}
	var cycles uint64 = 0
	for cycles < cyclesPerFrame {
		cycles += gba.CPU.Step()
	}
	// TODO: advance other components
}
