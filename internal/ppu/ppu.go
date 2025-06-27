package ppu

import (
	"github.com/Div9851/gba-go/internal/irq"
)

const (
	cyclesPerScanline = 1232
	cyclesPerPixel    = 4
	totalScanlines    = 228
	screenWidth       = 240
	screenHeight      = 160
)

type PPU struct {
	PRAM        [1024]byte
	VRAM        [96 * 1024]byte
	OAM         [1024]byte
	DISPCNT     uint16
	DISPSTAT    uint16
	VCOUNT      uint16
	cycles      uint64
	frameBuffer [screenHeight * screenWidth * 4]byte
	IRQ         *irq.IRQ
}

func NewPPU(irq *irq.IRQ) *PPU {
	return &PPU{
		IRQ: irq,
	}
}

func (ppu *PPU) Step() {
	ppu.cycles++

	if ppu.cycles >= cyclesPerScanline {
		ppu.cycles -= cyclesPerScanline

		if ppu.VCOUNT < screenHeight {
			ppu.RenderScanline()
		}
		ppu.VCOUNT += 1
		if ppu.VCOUNT >= totalScanlines {
			ppu.VCOUNT = 0
		}
	}

	ppu.UpdateDispStat()
}

func (ppu *PPU) RenderScanline() {
	bgMode := ppu.DISPCNT & 0x7
	switch bgMode {
	case 0x3:
		ppu.RenderMode3Scanline()
	case 0x4:
		ppu.RenderMode4Scanline()
	}
}

func (ppu *PPU) RenderMode3Scanline() {
	y := int(ppu.VCOUNT)
	for x := 0; x < screenWidth; x++ {
		addr := (y*screenWidth + x) * 2
		pixel := uint16(ppu.VRAM[addr]) | (uint16(ppu.VRAM[addr+1]) << 8)

		r := uint8((pixel & 0x1F) << 3)
		g := uint8(((pixel >> 5) & 0x1F) << 3)
		b := uint8(((pixel >> 10) & 0x1F) << 3)

		ppu.frameBuffer[(y*screenWidth+x)*4] = r
		ppu.frameBuffer[(y*screenWidth+x)*4+1] = g
		ppu.frameBuffer[(y*screenWidth+x)*4+2] = b
		ppu.frameBuffer[(y*screenWidth+x)*4+3] = 0xFF
	}
}

func (ppu *PPU) RenderMode4Scanline() {
	y := int(ppu.VCOUNT)
	for x := 0; x < screenWidth; x++ {
		addr := y*screenWidth + x
		index := ppu.VRAM[addr]
		color := uint16(ppu.PRAM[index*2]) | (uint16(ppu.PRAM[index*2+1]) << 8)

		r := uint8((color & 0x1F) * 255 / 31)
		g := uint8(((color >> 5) & 0x1F) * 255 / 31)
		b := uint8(((color >> 10) & 0x1F) * 255 / 31)

		ppu.frameBuffer[(y*screenWidth+x)*4] = r
		ppu.frameBuffer[(y*screenWidth+x)*4+1] = g
		ppu.frameBuffer[(y*screenWidth+x)*4+2] = b
		ppu.frameBuffer[(y*screenWidth+x)*4+3] = 0xFF
	}
}

func (ppu *PPU) UpdateDispStat() {
	if ppu.VCOUNT >= screenHeight { // VBLANK
		if (ppu.DISPSTAT&0x1) == 0 && (ppu.DISPSTAT&(1<<3)) != 0 {
			ppu.IRQ.IF |= 0x1
		}
		ppu.DISPSTAT |= 0x1
	} else {
		ppu.DISPSTAT &= 0xFFFE
	}

	if ppu.cycles >= cyclesPerPixel*screenWidth { // HBLANK
		if (ppu.DISPSTAT&0x2) == 0 && (ppu.DISPSTAT&(1<<4)) != 0 {
			ppu.IRQ.IF |= 0x2
		}
		ppu.DISPSTAT |= 0x2
	} else {
		ppu.DISPSTAT &= 0xFFFD
	}

	vcount := (ppu.DISPSTAT >> 8) & 0xFF
	if ppu.VCOUNT == vcount {
		if (ppu.DISPSTAT&0x4) == 0 && (ppu.DISPSTAT&(1<<5)) != 0 {
			ppu.IRQ.IF |= 0x4
		}
		ppu.DISPSTAT |= 0x4
	} else {
		ppu.DISPSTAT &= 0xFFFB
	}
}

func (ppu *PPU) GetFrameBuffer() []byte {
	return ppu.frameBuffer[:]
}
