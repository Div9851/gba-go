package gamepak

type GamePak struct {
	ROM  [96 * 1024 * 1024]byte
	SRAM [64 * 1024]byte
}

func NewGamePak() *GamePak {
	return &GamePak{}
}
