package gamepak

import (
	"bytes"
)

type GamePak struct {
	ROM    [32 * 1024 * 1024]byte
	Backup BackupDevice
}

func GetBackupDevice(data []byte) BackupDevice {
	if bytes.Contains(data, []byte("SRAM")) {
		return &SRAM{}
	} else if bytes.Contains(data, []byte("FLASH1M")) {
		return &Flash128K{}
	}
	// fallback
	return &SRAM{}
}

type BackupDevice interface {
	Read8(addr uint32) byte
	Write8(addr uint32, value byte)
}

type SRAM struct {
	data [32 * 1024]byte
}

func (sram *SRAM) Read8(addr uint32) byte {
	return sram.data[addr&0x7FFF]
}

func (sram *SRAM) Write8(addr uint32, value byte) {
	sram.data[addr&0x7FFF] = value
}

const (
	None = iota
	EnterIDMode1
	EnterIOMode2
	IOMode
	TerminateIOMode1
	TerminateIOMode2
)

type Flash128K struct {
	data  [128 * 1024]byte
	state int
}

func (flush *Flash128K) Read8(addr uint32) byte {
	if EnterIDMode1 <= flush.state && flush.state <= IOMode {
		// Sanyo
		if addr == 0x0 { // man
			return 0x62
		}
		if addr == 0x1 { // dev
			return 0x13
		}
	}
	return flush.data[addr]
}

func (flash *Flash128K) Write8(addr uint32, value byte) {
	switch flash.state {
	case None:
		if addr == 0x5555 && value == 0xAA {
			flash.state = EnterIDMode1
		}
	case EnterIDMode1:
		if addr == 0x2AAA && value == 0x55 {
			flash.state = EnterIOMode2
		} else {
			flash.state = None
		}
	case EnterIOMode2:
		if addr == 0x5555 && value == 0x90 {
			flash.state = IOMode
		} else {
			flash.state = None
		}
	case IOMode:
		if addr == 0x5555 && value == 0xAA {
			flash.state = TerminateIOMode1
		}
	case TerminateIOMode1:
		if addr == 0x2AAA && value == 0x55 {
			flash.state = TerminateIOMode2
		} else {
			flash.state = IOMode
		}
	case TerminateIOMode2:
		if addr == 0x5555 && value == 0xF0 {
			flash.state = None
		} else {
			flash.state = IOMode
		}
	}
	flash.data[addr] = value
}

func NewGamePak(data []byte) *GamePak {
	gamepak := &GamePak{}
	copy(gamepak.ROM[:], data)
	gamepak.Backup = GetBackupDevice(data)
	return gamepak
}
