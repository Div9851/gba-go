package input

import "github.com/Div9851/gba-go/internal/irq"

const (
	ButtonA uint16 = 1
	ButtonB uint16 = 1 << 1
	Select  uint16 = 1 << 2
	Start   uint16 = 1 << 3
	Right   uint16 = 1 << 4
	Left    uint16 = 1 << 5
	Up      uint16 = 1 << 6
	Down    uint16 = 1 << 7
	ButtonR uint16 = 1 << 8
	ButtonL uint16 = 1 << 9
)

type Input struct {
	KEYINPUT uint16
	KEYCNT   uint16
	IRQ      *irq.IRQ
}

func NewInput(irq *irq.IRQ) *Input {
	return &Input{
		KEYINPUT: 0xFFFF,
		IRQ:      irq,
	}
}

func (input *Input) SetKeys(keys []string) {
	var keyInput uint16 = 0xFFFF

	for _, key := range keys {
		switch key {
		case "ArrowRight":
			keyInput &= ^Right
		case "ArrowLeft":
			keyInput &= ^Left
		case "ArrowUp":
			keyInput &= ^Up
		case "ArrowDown":
			keyInput &= ^Down
		case "A":
			keyInput &= ^ButtonL
		case "S":
			keyInput &= ^ButtonR
		case "X":
			keyInput &= ^ButtonA
		case "Z":
			keyInput &= ^ButtonB
		case "Enter":
			keyInput &= ^Start
		case "Backspace":
			keyInput &= ^Select
		}
	}

	if (^keyInput & input.KEYINPUT & input.KEYCNT) != 0 {
		input.IRQ.IF |= 1 << 12
	}

	input.KEYINPUT = keyInput
}
