package ppu

type PPU struct {
	PRAM [1024]byte
	VRAM [96 * 1024]byte
	OAM  [1024]byte
}

func NewPPU() *PPU {
	return &PPU{}
}
