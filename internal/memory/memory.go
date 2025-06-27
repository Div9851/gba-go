package memory

type Memory interface {
	Read8(addr uint32) byte
	Write8(addr uint32, value byte)
	Read16(addr uint32) uint16
	Write16(addr uint32, value uint16)
	Read32(addr uint32) uint32
	Write32(addr uint32, value uint32)
}
