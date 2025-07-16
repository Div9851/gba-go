package ppu

import (
	"github.com/Div9851/gba-go/internal/dma"
	"github.com/Div9851/gba-go/internal/irq"
)

const (
	cyclesPerScanline = 1232
	cyclesPerPixel    = 4
	totalScanlines    = 228
	screenWidth       = 240
	screenHeight      = 160
)

type Color struct {
	R           byte
	G           byte
	B           byte
	Transparent bool
}

type Pixel struct {
	Color    Color
	Priority int
	Layer    int // 0-3: BG 4: OBJ
	Valid    bool
}

type OAMEntry struct {
	Y         int
	X         int
	TileIndex int
	Priority  int
	Palette   int
	Width     int
	Height    int
	HFlip     bool
	VFlip     bool
	Use256    bool
	Visible   bool
}

type PPU struct {
	PRAM        [1024]byte
	VRAM        [96 * 1024]byte
	OAM         [1024]byte
	DISPCNT     uint16
	DISPSTAT    uint16
	VCOUNT      uint16
	BGCNT       [4]uint16
	BGHOFS      [4]uint16
	BGVOFS      [4]uint16
	cycles      uint64
	frameBuffer [screenHeight * screenWidth * 4]byte
	IRQ         *irq.IRQ
	DMA         [4]*dma.Channel
}

var textBGSizes = [4][2]int{
	{256, 256},
	{512, 256},
	{256, 512},
	{512, 512},
}

var spriteSizes = [4][4][2]int{
	// Square
	{{8, 8}, {16, 16}, {32, 32}, {64, 64}},
	// Horizontal
	{{16, 8}, {32, 8}, {32, 16}, {64, 32}},
	// Vertical
	{{8, 16}, {8, 32}, {16, 32}, {32, 64}},
	// Invalid
	{{0, 0}, {0, 0}, {0, 0}, {0, 0}},
}

func NewPPU(irq *irq.IRQ, dma [4]*dma.Channel) *PPU {
	return &PPU{
		IRQ: irq,
		DMA: dma,
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

func (ppu *PPU) GetColor(paletteIndex int, paletteBaseAddr int) Color {
	value := uint16(ppu.PRAM[paletteBaseAddr+paletteIndex*2]) | (uint16(ppu.PRAM[paletteBaseAddr+paletteIndex*2+1]) << 8)
	r := byte((value & 0x1F) * 255 / 31)
	g := byte(((value >> 5) & 0x1F) * 255 / 31)
	b := byte(((value >> 10) & 0x1F) * 255 / 31)
	return Color{
		R: r,
		G: g,
		B: b,
	}
}

func (ppu *PPU) GetTilePixel(tileDataAddr int, x int, y int, palette int, paletteBaseAddr int, use256 bool) Color {
	var tileColorIndex int

	if use256 {
		tileColorIndex = int(ppu.VRAM[tileDataAddr+(y*8+x)])
	} else {
		tileColorIndex = int((ppu.VRAM[tileDataAddr+(y*4+x/2)] >> (4 * (x % 2))) & 0xF)
	}

	if tileColorIndex == 0 {
		return Color{Transparent: true}
	}

	var paletteIndex int
	if use256 {
		paletteIndex = tileColorIndex
	} else {
		paletteIndex = palette*16 + tileColorIndex
	}

	return ppu.GetColor(paletteIndex, paletteBaseAddr)
}

func (ppu *PPU) GetOBJTileDataAddr(tileIndex int, x int, y int, spriteWidth int, use256 bool) int {
	objTileBase := 0x10000

	is1DMapping := (ppu.DISPCNT & (1 << 6)) != 0

	var tileDataAddr int
	var tileSize int

	if use256 {
		tileSize = 64
	} else {
		tileSize = 32
	}

	if is1DMapping {
		currentTile := tileIndex + (y/8)*(spriteWidth/8) + (x / 8)
		tileDataAddr = objTileBase + currentTile*tileSize
	} else {
		tileX := tileIndex%32 + (x / 8)
		tileY := tileIndex/32 + (y / 8)
		currentTile := tileY*32 + tileX
		tileDataAddr = objTileBase + currentTile*tileSize
	}

	return tileDataAddr
}

func (ppu *PPU) ParseOAMEntry(index int) OAMEntry {
	attr0 := uint16(ppu.OAM[index*8+1])<<8 | uint16(ppu.OAM[index*8])
	attr1 := uint16(ppu.OAM[index*8+3])<<8 | uint16(ppu.OAM[index*8+2])
	attr2 := uint16(ppu.OAM[index*8+5])<<8 | uint16(ppu.OAM[index*8+4])

	shape := int((attr0 >> 14) & 0x3)
	size := int((attr1 >> 14) & 0x3)
	spriteSize := spriteSizes[shape][size]

	return OAMEntry{
		Y:         int(attr0 & 0xFF),
		X:         int(attr1 & 0x1FF),
		TileIndex: int(attr2 & 0x3FF),
		Priority:  int((attr2 >> 10) & 0x3),
		Palette:   int((attr2 >> 12) & 0xF),
		Width:     spriteSize[0],
		Height:    spriteSize[1],
		HFlip:     (attr1 & (1 << 12)) != 0,
		VFlip:     (attr1 & (1 << 13)) != 0,
		Use256:    (attr0 & (1 << 13)) != 0,
		Visible:   (attr0 & (1 << 9)) == 0,
	}
}

func (ppu *PPU) RenderPixel(pixels [][screenWidth]Pixel, x int, y int) {
	var finalPixel Pixel
	highestPriority := 4
	found := false

	if pixels[4][x].Valid && !pixels[4][x].Color.Transparent {
		finalPixel = pixels[4][x]
		highestPriority = pixels[4][x].Priority
		found = true
	}

	for layer := 0; layer < 4; layer++ {
		pixel := pixels[layer][x]
		if pixel.Valid && !pixel.Color.Transparent {
			if !found || pixel.Priority < highestPriority {
				finalPixel = pixel
				highestPriority = pixel.Priority
				found = true
			}
		}
	}

	if !found {
		value := uint16(ppu.PRAM[0]) | (uint16(ppu.PRAM[1]) << 8)
		r := byte((value & 0x1F) * 255 / 31)
		g := byte(((value >> 5) & 0x1F) * 255 / 31)
		b := byte(((value >> 10) & 0x1F) * 255 / 31)
		finalPixel.Color = Color{
			R: r,
			G: g,
			B: b,
		}
	}

	ppu.frameBuffer[(y*screenWidth+x)*4] = finalPixel.Color.R
	ppu.frameBuffer[(y*screenWidth+x)*4+1] = finalPixel.Color.G
	ppu.frameBuffer[(y*screenWidth+x)*4+2] = finalPixel.Color.B
	ppu.frameBuffer[(y*screenWidth+x)*4+3] = 0xFF
}

func (ppu *PPU) RenderScanline() {
	bgMode := ppu.DISPCNT & 0x7
	y := int(ppu.VCOUNT)
	var pixels [5][screenWidth]Pixel
	switch bgMode {
	case 0x0:
		ppu.RenderMode0Scanline(pixels[:], y)
	case 0x3:
		ppu.RenderMode3Scanline(pixels[:], y)
	case 0x4:
		ppu.RenderMode4Scanline(pixels[:], y)
	}
	ppu.RenderOBJScanline(pixels[:], y)
	for x := 0; x < screenWidth; x++ {
		ppu.RenderPixel(pixels[:], x, y)
	}
}

func (ppu *PPU) RenderTextBGScanline(pixels [][screenWidth]Pixel, bgIndex int, y int) {
	if ppu.DISPCNT&(1<<(8+bgIndex)) == 0 {
		return
	}

	bgSize := (ppu.BGCNT[bgIndex] >> 14) & 0x3
	bgWidth, bgHeight := textBGSizes[bgSize][0], textBGSizes[bgSize][1]

	use256 := (ppu.BGCNT[bgIndex] & (1 << 7)) != 0
	var tileSize int
	if use256 {
		tileSize = 64
	} else {
		tileSize = 32
	}

	tileDataBaseAddr := int((ppu.BGCNT[bgIndex]>>2)&0x3) * (16 * 1024)
	tileMapBaseAddr := int((ppu.BGCNT[bgIndex]>>8)&0x1F) * (2 * 1024)

	bgY := (y + int(ppu.BGVOFS[bgIndex]&0x1FF)) % bgHeight
	for x := 0; x < screenWidth; x++ {
		bgX := (x + int(ppu.BGHOFS[bgIndex]&0x1FF)) % bgWidth
		if bgX >= bgWidth {
			return
		}
		tileMapAddr := tileMapBaseAddr
		areaX, areaY := bgX, bgY
		switch bgSize {
		case 1:
			tileMapAddr += (areaX / 256) * 2048
			areaX %= 256
		case 2:
			tileMapAddr += (areaY / 256) * 2048
			areaY %= 256
		case 3:
			tileMapAddr += (areaY/256)*4096 + (areaX/256)*2048
			areaX %= 256
			areaY %= 256
		}
		tileMapIndex := (areaY/8)*32 + (areaX / 8)
		tileMapAddr += tileMapIndex * 2
		tileMap := uint16(ppu.VRAM[tileMapAddr+1])<<8 | uint16(ppu.VRAM[tileMapAddr])

		tileDataIndex := int(tileMap & 0x3FF)
		tileDataAddr := tileDataBaseAddr + tileDataIndex*tileSize

		tileX := bgX % 8
		tileY := bgY % 8
		if (tileMap & (1 << 10)) != 0 { // Horizontal Flip
			tileX = 7 - tileX
		}
		if (tileMap & (1 << 11)) != 0 { // Vertical Flip
			tileY = 7 - tileY
		}

		palette := int((tileMap >> 12) & 0xF)

		pixels[bgIndex][x] = Pixel{
			Color:    ppu.GetTilePixel(tileDataAddr, tileX, tileY, palette, 0, use256),
			Priority: int(ppu.BGCNT[bgIndex] & 0x3),
			Layer:    bgIndex,
			Valid:    true,
		}
	}
}

func (ppu *PPU) RenderMode0Scanline(pixels [][screenWidth]Pixel, y int) {
	for bgIndex := 0; bgIndex < 4; bgIndex++ {
		ppu.RenderTextBGScanline(pixels, bgIndex, y)
	}
}

func (ppu *PPU) RenderMode3Scanline(pixels [][screenWidth]Pixel, y int) {
	for x := 0; x < screenWidth; x++ {
		addr := (y*screenWidth + x) * 2
		value := (uint16(ppu.VRAM[addr+1]) << 8) | uint16(ppu.VRAM[addr])
		r := byte((value & 0x1F) * 255 / 31)
		g := byte(((value >> 5) & 0x1F) * 255 / 31)
		b := byte(((value >> 10) & 0x1F) * 255 / 31)
		pixels[2][x] = Pixel{
			Color: Color{
				R: r,
				G: g,
				B: b,
			},
			Layer: 2,
			Valid: true,
		}
	}
}

func (ppu *PPU) RenderMode4Scanline(pixels [][screenWidth]Pixel, y int) {
	for x := 0; x < screenWidth; x++ {
		addr := y*screenWidth + x
		paletteIndex := int(ppu.VRAM[addr])
		pixels[2][x] = Pixel{
			Color: ppu.GetColor(paletteIndex, 0),
			Layer: 2,
			Valid: true,
		}
	}
}

func (ppu *PPU) RenderOBJScanline(pixels [][screenWidth]Pixel, y int) {
	if (ppu.DISPCNT & (1 << 12)) == 0 {
		return
	}

	var entries []OAMEntry
	for index := 0; index < 128; index++ {
		entries = append(entries, ppu.ParseOAMEntry(index))
	}

	for _, entry := range entries {
		ppu.RenderSprite(pixels, entry, y)
	}
}

func (ppu *PPU) RenderSprite(pixels [][screenWidth]Pixel, entry OAMEntry, y int) {
	if !entry.Visible || y < entry.Y || y >= entry.Y+entry.Height {
		return
	}

	spriteY := y - entry.Y
	if entry.VFlip {
		spriteY = entry.Height - 1 - spriteY
	}

	for spriteX := 0; spriteX < entry.Width; spriteX++ {
		screenX := entry.X + spriteX
		if screenX >= screenWidth {
			break
		}

		actualSpriteX := spriteX
		if entry.HFlip {
			actualSpriteX = entry.Width - 1 - spriteX
		}

		tileDataAddr := ppu.GetOBJTileDataAddr(entry.TileIndex, actualSpriteX, spriteY, entry.Width, entry.Use256)
		color := ppu.GetTilePixel(tileDataAddr, actualSpriteX%8, spriteY%8, entry.Palette, 0x200, entry.Use256)

		if !color.Transparent {
			currentPixel := pixels[4][screenX]
			if !currentPixel.Valid || currentPixel.Color.Transparent || entry.Priority < currentPixel.Priority {
				pixels[4][screenX] = Pixel{
					Color:    color,
					Priority: entry.Priority,
					Layer:    4,
					Valid:    true,
				}
			}
		}
	}
}

func (ppu *PPU) UpdateDispStat() {
	if ppu.VCOUNT >= screenHeight { // VBLANK
		if (ppu.DISPSTAT & 0x1) == 0 {
			if (ppu.DISPSTAT & (1 << 3)) != 0 {
				ppu.IRQ.IF |= 0x1
			}

			for ch := 0; ch < 4; ch++ {
				if ppu.DMA[ch].Status == dma.Wait && ppu.DMA[ch].Cond == dma.VBlank {
					ppu.DMA[ch].Trigger()
				}
			}
		}
		ppu.DISPSTAT |= 0x1
	} else {
		ppu.DISPSTAT &= 0xFFFE
	}

	if ppu.cycles >= cyclesPerPixel*screenWidth { // HBLANK
		if (ppu.DISPSTAT & 0x2) == 0 {
			if (ppu.DISPSTAT & (1 << 4)) != 0 {
				ppu.IRQ.IF |= 0x2
			}

			for ch := 0; ch < 4; ch++ {
				if ppu.DMA[ch].Status == dma.Wait && ppu.DMA[ch].Cond == dma.HBlank {
					ppu.DMA[ch].Trigger()
				}
			}
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
