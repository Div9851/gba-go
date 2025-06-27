package main

import (
	"log"

	"github.com/Div9851/gba-go/pkg/emulator"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	// GBA screen dimensions
	screenWidth  = 240
	screenHeight = 160
	// Scale factor for display
	scaleFactor = 3
)

type Game struct {
	emulator *emulator.GBA
}

func (g *Game) Update() error {
	g.emulator.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "GBA Emulator - Under Development")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	gba := emulator.NewGBA()

	game := &Game{
		emulator: gba,
	}

	ebiten.SetWindowSize(screenWidth*scaleFactor, screenHeight*scaleFactor)
	ebiten.SetWindowTitle("GBA Emulator")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
