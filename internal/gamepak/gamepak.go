package gamepak

import (
	"bytes"
)

const (
	EEPROM = iota
	SRAM
	FLASH64KB
	FLASH128KB
)

type GamePak struct {
	ROM        [32 * 1024 * 1024]byte
	SRAM       [64 * 1024]byte
	BackupType int
}

func GetBackupType(data []byte) int {
	if bytes.Contains(data, []byte("EEPROM")) {
		return EEPROM
	} else if bytes.Contains(data, []byte("SRAM")) {
		return SRAM
	} else if bytes.Contains(data, []byte("FLASH1M")) {
		return FLASH128KB
	}
	return FLASH64KB
}

func NewGamePak(data []byte) *GamePak {
	gamepak := &GamePak{}
	copy(gamepak.ROM[:], data)
	gamepak.BackupType = GetBackupType(data)
	return gamepak
}
