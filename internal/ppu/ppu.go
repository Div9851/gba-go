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
	Y              int
	X              int
	TileIndex      int
	Priority       int
	Palette        int
	Width          int
	Height         int
	Use256         bool
	UseRotateScale bool

	// Rotate/Scale used
	DoubleSize bool
	PA         int16
	PB         int16
	PC         int16
	PD         int16

	// Rotate/Scale not used
	HFlip  bool
	VFlip  bool
	Enable bool
}

func (entry *OAMEntry) GetRectWidth() int {
	if entry.UseRotateScale && entry.DoubleSize {
		return entry.Width * 2
	}
	return entry.Width
}

func (entry *OAMEntry) GetRectHeight() int {
	if entry.UseRotateScale && entry.DoubleSize {
		return entry.Height * 2
	}
	return entry.Height
}

func (entry *OAMEntry) GetCoordinate(rectX int, rectY int) (spX int, spY int) {
	if entry.UseRotateScale {
		dx := rectX - entry.GetRectWidth()/2
		dy := rectY - entry.GetRectHeight()/2
		dx88, dy88 := int32(dx<<8), int32(dy<<8)
		u88 := int32(entry.PA)*dx88 + int32(entry.PB)*dy88
		v88 := int32(entry.PC)*dx88 + int32(entry.PD)*dy88
		spX = int(u88>>16) + entry.Width/2
		spY = int(v88>>16) + entry.Height/2
		return
	} else {
		spX = rectX
		if entry.HFlip {
			spX = entry.Width - 1 - spX
		}
		spY = rectY
		if entry.VFlip {
			spY = entry.Height - 1 - spY
		}
		return
	}
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
	BGX_L       [4]uint16
	BGX_H       [4]uint16
	BGY_L       [4]uint16
	BGY_H       [4]uint16
	BG_PA       [4]uint16
	BG_PB       [4]uint16
	BG_PC       [4]uint16
	BG_PD       [4]uint16
	cycles      uint64
	frameBuffer [screenHeight * screenWidth * 4]byte
	OAMEntries  []*OAMEntry
	IRQ         *irq.IRQ
	DMA         [4]*dma.Channel
}

var textBGSizes = [4][2]int{
	{256, 256},
	{512, 256},
	{256, 512},
	{512, 512},
}

var rotScaleBGSizes = [4][2]int{
	{128, 128},
	{256, 256},
	{512, 512},
	{1024, 1024},
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

		if ppu.VCOUNT == 0 {
			ppu.LoadOAMEntries()
		}

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
	var objTileBase int
	if (ppu.DISPCNT & 0x7) >= 3 { // bitmap mode
		objTileBase = 0x14000
	} else { // non-bitmap mode
		objTileBase = 0x10000
	}

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

func (ppu *PPU) LoadOAMEntries() {
	ppu.OAMEntries = ppu.OAMEntries[:0]
	for index := 0; index < 128; index++ {
		ppu.OAMEntries = append(ppu.OAMEntries, ppu.ParseOAMEntry(index))
	}

}

func (ppu *PPU) ParseOAMEntry(index int) *OAMEntry {
	attr0 := uint16(ppu.OAM[index*8+1])<<8 | uint16(ppu.OAM[index*8])
	attr1 := uint16(ppu.OAM[index*8+3])<<8 | uint16(ppu.OAM[index*8+2])
	attr2 := uint16(ppu.OAM[index*8+5])<<8 | uint16(ppu.OAM[index*8+4])

	y := int(attr0 & 0xFF)
	if y >= screenHeight {
		y -= 256
	}

	x := int(attr1 & 0x1FF)
	if x >= screenWidth {
		x -= 512
	}

	use256 := (attr0 & (1 << 13)) != 0
	tileIndex := int(attr2 & 0x3FF)
	if use256 {
		tileIndex /= 2
	}
	priority := int((attr2 >> 10) & 0x3)
	palette := int((attr2 >> 12) & 0xF)
	shape := int((attr0 >> 14) & 0x3)
	size := int((attr1 >> 14) & 0x3)
	spriteSize := spriteSizes[shape][size]

	useRotateScale := (attr0 & (1 << 8)) != 0

	if useRotateScale {
		paramAddr := 0x6 + int((attr1>>9)&0x1F)*0x20
		pa := int16(uint16(ppu.OAM[paramAddr+0x1])<<8 | uint16(ppu.OAM[paramAddr]))
		pb := int16(uint16(ppu.OAM[paramAddr+0x9])<<8 | uint16(ppu.OAM[paramAddr+0x8]))
		pc := int16(uint16(ppu.OAM[paramAddr+0x11])<<8 | uint16(ppu.OAM[paramAddr+0x10]))
		pd := int16(uint16(ppu.OAM[paramAddr+0x19])<<8 | uint16(ppu.OAM[paramAddr+0x18]))
		return &OAMEntry{
			Y:              y,
			X:              x,
			TileIndex:      tileIndex,
			Priority:       priority,
			Palette:        palette,
			Width:          spriteSize[0],
			Height:         spriteSize[1],
			Use256:         use256,
			UseRotateScale: useRotateScale,
			DoubleSize:     (attr0 & (1 << 9)) != 0,
			PA:             pa,
			PB:             pb,
			PC:             pc,
			PD:             pd,
		}
	} else {
		return &OAMEntry{
			Y:              y,
			X:              x,
			TileIndex:      tileIndex,
			Priority:       priority,
			Palette:        palette,
			Width:          spriteSize[0],
			Height:         spriteSize[1],
			Use256:         use256,
			UseRotateScale: useRotateScale,
			HFlip:          (attr1 & (1 << 12)) != 0,
			VFlip:          (attr1 & (1 << 13)) != 0,
			Enable:         (attr0 & (1 << 9)) == 0,
		}
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
	case 0x1:
		ppu.RenderMode1Scanline(pixels[:], y)
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

	bgY := (y + int(ppu.BGVOFS[bgIndex]&0x1FF)) & (bgHeight - 1)
	for x := 0; x < screenWidth; x++ {
		bgX := (x + int(ppu.BGHOFS[bgIndex]&0x1FF)) & (bgWidth - 1)
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

func (ppu *PPU) RenderRotScaleBGScanline(pixels [][screenWidth]Pixel, bgIndex int, y int) {
	if ppu.DISPCNT&(1<<(8+bgIndex)) == 0 {
		return
	}

	bgSize := (ppu.BGCNT[bgIndex] >> 14) & 0x3
	bgWidth, bgHeight := rotScaleBGSizes[bgSize][0], rotScaleBGSizes[bgSize][1]

	tileDataBaseAddr := int((ppu.BGCNT[bgIndex]>>2)&0x3) * (16 * 1024)
	tileMapBaseAddr := int((ppu.BGCNT[bgIndex]>>8)&0x1F) * (2 * 1024)

	pa := int32(int16(ppu.BG_PA[bgIndex]))
	pb := int32(int16(ppu.BG_PB[bgIndex]))
	pc := int32(int16(ppu.BG_PC[bgIndex]))
	pd := int32(int16(ppu.BG_PD[bgIndex]))
	baseX := uint32(ppu.BGX_H[bgIndex]&0xFFF)<<16 | uint32(ppu.BGX_L[bgIndex])
	if baseX&(1<<27) != 0 {
		baseX |= 0xF0000000
	}
	baseY := uint32(ppu.BGY_H[bgIndex]&0xFFF)<<16 | uint32(ppu.BGY_L[bgIndex])
	if baseY&(1<<27) != 0 {
		baseY |= 0xF0000000
	}
	u := pb*int32(y) + int32(baseX)
	v := pd*int32(y) + int32(baseY)
	for x := 0; x < screenWidth; x++ {
		bgX := int(u>>8) & (bgWidth - 1)
		bgY := int(v>>8) & (bgHeight - 1)

		tileMapIndex := (bgY/8)*(bgWidth/8) + (bgX / 8)
		tileMapAddr := tileMapBaseAddr + tileMapIndex
		tileDataIndex := int(ppu.VRAM[tileMapAddr])
		tileSize := 64
		tileDataAddr := tileDataBaseAddr + tileDataIndex*tileSize

		tileX := bgX % 8
		tileY := bgY % 8

		pixels[bgIndex][x] = Pixel{
			Color:    ppu.GetTilePixel(tileDataAddr, tileX, tileY, 0, 0, true),
			Priority: int(ppu.BGCNT[bgIndex] & 0x3),
			Layer:    bgIndex,
			Valid:    true,
		}

		u += pa
		v += pc
	}
}

func (ppu *PPU) RenderMode0Scanline(pixels [][screenWidth]Pixel, y int) {
	for bgIndex := 0; bgIndex < 4; bgIndex++ {
		ppu.RenderTextBGScanline(pixels, bgIndex, y)
	}
}

func (ppu *PPU) RenderMode1Scanline(pixels [][screenWidth]Pixel, y int) {
	for bgIndex := 0; bgIndex < 2; bgIndex++ {
		ppu.RenderTextBGScanline(pixels, bgIndex, y)
	}
	ppu.RenderRotScaleBGScanline(pixels, 2, y)
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

	for _, entry := range ppu.OAMEntries {
		ppu.RenderSprite(pixels, entry, y)
	}
}

func (ppu *PPU) RenderSprite(pixels [][screenWidth]Pixel, entry *OAMEntry, y int) {
	if !entry.UseRotateScale && !entry.Enable {
		return
	}
	if y < entry.Y || y >= entry.Y+entry.GetRectHeight() {
		return
	}

	rectY := y - entry.Y
	for rectX := 0; rectX < entry.GetRectWidth(); rectX++ {
		screenX := entry.X + rectX
		if screenX < 0 || screenX >= screenWidth {
			continue
		}

		spX, spY := entry.GetCoordinate(rectX, rectY)
		if spX < 0 || spX >= entry.Width || spY < 0 || spY >= entry.Height {
			continue
		}

		tileDataAddr := ppu.GetOBJTileDataAddr(entry.TileIndex, spX, spY, entry.Width, entry.Use256)
		color := ppu.GetTilePixel(tileDataAddr, spX%8, spY%8, entry.Palette, 0x200, entry.Use256)

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
