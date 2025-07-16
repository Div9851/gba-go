package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Div9851/gba-go/pkg/emulator"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// GBA screen dimensions
	screenWidth  = 240
	screenHeight = 160
	// Scale factor for display
	scaleFactor = 2
)

type Game struct {
	emulator *emulator.GBA
	keys     []ebiten.Key
}

func (g *Game) Update() error {
	g.keys = inpututil.AppendPressedKeys(g.keys[:0])
	var keys []string
	for _, key := range g.keys {
		keys = append(keys, key.String())
	}
	g.emulator.Update(keys)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.WritePixels(g.emulator.PPU.GetFrameBuffer())
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	var (
		biosFilePath = flag.String("bios", "assets/bios.bin", "BIOS file path")
		romFilePath  = flag.String("rom", "assets/hello.gba", "ROM file path")
		debug        = flag.Bool("debug", false, "debug mode")
	)

	flag.Parse()

	gba := emulator.NewGBA()

	biosData, err := os.ReadFile(*biosFilePath)
	if err != nil {
		panic(err)
	}
	gba.LoadBIOS(biosData)

	romData, err := os.ReadFile(*romFilePath)
	if err != nil {
		panic(err)
	}
	gba.LoadROM(romData)

	gba.Start()

	if *debug {
		stepCount := 0
		sc := bufio.NewScanner(os.Stdin)
		breakpoints := []uint32{}
		for {
			fmt.Printf("step %d\n", stepCount)
			opcode := gba.CPU.Pipeline[1]
			if gba.CPU.IsThumb() {
				fmt.Printf("%08X: %04X\n\n", gba.CPU.ReadReg(15)-4, opcode)
			} else {
				fmt.Printf("%08X: %08X\n\n", gba.CPU.ReadReg(15)-8, opcode)
			}

			sc.Scan()
			inputs := strings.Split(sc.Text(), " ")
			switch inputs[0] {
			case "break":
				addr, _ := strconv.ParseInt(inputs[1], 16, 32)
				breakpoints = append(breakpoints, uint32(addr))
			case "continue":
				for {
					pc := gba.CPU.ReadReg(15)
					if gba.CPU.IsThumb() {
						pc -= 4
					} else {
						pc -= 8
					}
					if slices.Contains(breakpoints, pc) {
						break
					}
					gba.Step()
					stepCount++
				}
			case "nextN":
				nn, _ := strconv.Atoi(inputs[1])
				for i := 0; i < nn; i++ {
					gba.Step()
				}
				stepCount += nn
			default:
				gba.Step()
				stepCount++
			}

			for i := 0; i <= 15; i++ {
				fmt.Printf("R%d: %08X ", i, gba.CPU.ReadReg(i))
			}
			fmt.Printf("CPSR: %08X\n\n", gba.CPU.CPSR)
		}
	}

	game := &Game{
		emulator: gba,
	}

	ebiten.SetWindowSize(screenWidth*scaleFactor, screenHeight*scaleFactor)
	ebiten.SetWindowTitle("GBA Emulator")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
