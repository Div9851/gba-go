package cpu

import (
	"math/bits"

	"github.com/Div9851/gba-go/internal/bus"
	"github.com/Div9851/gba-go/internal/irq"
)

const (
	BitN uint32 = 1 << 31
	BitZ uint32 = 1 << 30
	BitC uint32 = 1 << 29
	BitV uint32 = 1 << 28
	BitI uint32 = 1 << 7
	BitF uint32 = 1 << 6
	BitT uint32 = 1 << 5
	BitM uint32 = 0x1F
)

const (
	ModeUSR = 0x10
	ModeFIQ = 0x11
	ModeIRQ = 0x12
	ModeSVC = 0x13
	ModeABT = 0x17
	ModeUND = 0x1B
	ModeSYS = 0x1F
)

const (
	ExceptReset = iota
	ExceptUndefined
	ExceptSoftwareInterrupt
	ExceptPrefetchAbort
	ExceptDataAbort
	ExceptAddressExceeds26bit
	ExceptNormalInterrupt
	ExceptFastInterrupt
)

type CPU struct {
	reg                 [16]uint32
	bankedReg           [5][16]uint32 // FIQ IRQ SVC ABT UND
	CPSR                uint32
	SPSR                [5]uint32
	Bus                 *bus.Bus
	IRQ                 *irq.IRQ
	Pipeline            [2]uint32
	ShouldResetPipeline bool
}

func NewCPU(bus *bus.Bus, irq *irq.IRQ) *CPU {
	reg := [16]uint32{}
	reg[13] = 0x03007F00
	reg[14] = 0x08000000
	reg[15] = 0x08000000
	bankedReg := [5][16]uint32{}
	bankedReg[2][13] = 0x03007F00
	return &CPU{
		reg:       reg,
		bankedReg: bankedReg,
		CPSR:      0x1F,
		Bus:       bus,
		IRQ:       irq,
	}
}

func (cpu *CPU) ReadReg(index int) uint32 {
	mode := cpu.Mode()
	if index < 8 || index == 15 {
		return cpu.reg[index]
	}
	if index < 13 {
		if mode == ModeFIQ {
			return cpu.bankedReg[0][index]
		}
		return cpu.reg[index]
	}
	switch mode {
	case ModeUSR, ModeSYS:
		return cpu.reg[index]
	case ModeFIQ:
		return cpu.bankedReg[0][index]
	case ModeIRQ:
		return cpu.bankedReg[1][index]
	case ModeSVC:
		return cpu.bankedReg[2][index]
	case ModeABT:
		return cpu.bankedReg[3][index]
	case ModeUND:
		return cpu.bankedReg[4][index]
	}
	return 0xFFFFFFFF
}

func (cpu *CPU) WriteReg(index int, val uint32) {
	mode := cpu.Mode()
	if index < 8 {
		cpu.reg[index] = val
		return
	}
	if index == 15 {
		cpu.reg[index] = val
		cpu.ShouldResetPipeline = true
		return
	}
	if index < 13 {
		if mode == ModeFIQ {
			cpu.bankedReg[0][index] = val
			return
		}
		cpu.reg[index] = val
		return
	}
	switch mode {
	case ModeUSR, ModeSYS:
		cpu.reg[index] = val
	case ModeFIQ:
		cpu.bankedReg[0][index] = val
	case ModeIRQ:
		cpu.bankedReg[1][index] = val
	case ModeSVC:
		cpu.bankedReg[2][index] = val
	case ModeABT:
		cpu.bankedReg[3][index] = val
	case ModeUND:
		cpu.bankedReg[4][index] = val
	}
}

func (cpu *CPU) WriteUserReg(index int, val uint32) {
	cpu.reg[index] = val
}

func (cpu *CPU) ReadUserReg(index int) uint32 {
	return cpu.reg[index]
}

func (cpu *CPU) ReadSPSR(mode int) uint32 {
	switch mode {
	case ModeFIQ:
		return cpu.SPSR[0]
	case ModeIRQ:
		return cpu.SPSR[1]
	case ModeSVC:
		return cpu.SPSR[2]
	case ModeABT:
		return cpu.SPSR[3]
	case ModeUND:
		return cpu.SPSR[4]
	}
	// User/System mode does not have SPSR
	return cpu.CPSR
}

func (cpu *CPU) WriteSPSR(mode int, val uint32) {
	switch mode {
	case ModeFIQ:
		cpu.SPSR[0] = val
	case ModeIRQ:
		cpu.SPSR[1] = val
	case ModeSVC:
		cpu.SPSR[2] = val
	case ModeABT:
		cpu.SPSR[3] = val
	case ModeUND:
		cpu.SPSR[4] = val
	}
}

func (cpu *CPU) IsThumb() bool {
	return (cpu.CPSR & BitT) != 0
}

func (cpu *CPU) Mode() int {
	return int(cpu.CPSR & BitM)
}

func (cpu *CPU) SetFlags(n, z, c, v bool) {
	cpu.CPSR &= ^(BitN | BitZ | BitC | BitV)
	if n {
		cpu.CPSR |= BitN
	}
	if z {
		cpu.CPSR |= BitZ
	}
	if c {
		cpu.CPSR |= BitC
	}
	if v {
		cpu.CPSR |= BitV
	}
}

func (cpu *CPU) GetFlags() (n, z, c, v bool) {
	n = (cpu.CPSR & BitN) != 0
	z = (cpu.CPSR & BitZ) != 0
	c = (cpu.CPSR & BitC) != 0
	v = (cpu.CPSR & BitV) != 0
	return
}

// UpdateArithmeticFlags updates N, Z, C, V flags for arithmetic operations
func (cpu *CPU) UpdateArithmeticFlags(result uint32, carry, overflow bool) {
	var flags uint32
	if (result & (1 << 31)) != 0 {
		flags |= BitN
	}
	if result == 0 {
		flags |= BitZ
	}
	if carry {
		flags |= BitC
	}
	if overflow {
		flags |= BitV
	}
	cpu.CPSR = (cpu.CPSR & ^(BitN | BitZ | BitC | BitV)) | flags
}

// UpdateLogicalFlags updates N, Z, C flags for logical operations
func (cpu *CPU) UpdateLogicalFlags(result uint32, carry bool) {
	var flags uint32
	if (result & (1 << 31)) != 0 {
		flags |= BitN
	}
	if result == 0 {
		flags |= BitZ
	}
	if carry {
		flags |= BitC
	}
	cpu.CPSR = (cpu.CPSR & ^(BitN | BitZ | BitC)) | flags
}

// CalculateOverflow calculates overflow flag for arithmetic operations
func CalculateOverflow(op1, op2, result uint32, isSubtraction bool) bool {
	sign1 := (op1 >> 31) & 1
	sign2 := (op2 >> 31) & 1
	signRes := (result >> 31) & 1

	if isSubtraction {
		return sign1 != sign2 && sign1 != signRes
	} else {
		return sign1 == sign2 && sign1 != signRes
	}
}

func (cpu *CPU) HandleException(except int) {
	pc := cpu.ReadReg(15) // PC+nn
	var mode int
	var addr uint32
	switch except {
	case ExceptReset:
		mode = ModeSVC
		addr = 0x00
	case ExceptUndefined:
		mode = ModeUND
		addr = 0x04
	case ExceptSoftwareInterrupt:
		mode = ModeSVC
		addr = 0x08
	case ExceptPrefetchAbort:
		mode = ModeABT
		addr = 0x0C
	case ExceptDataAbort:
		mode = ModeABT
		addr = 0x10
	case ExceptAddressExceeds26bit:
		mode = ModeSVC
		addr = 0x14
	case ExceptNormalInterrupt:
		mode = ModeIRQ
		addr = 0x18
	case ExceptFastInterrupt:
		mode = ModeFIQ
		addr = 0x1C
	}
	cpsr := cpu.CPSR
	cpu.CPSR = (cpu.CPSR & ^BitM) | uint32(mode) // set mode
	cpu.CPSR |= BitI                             // IRQs diabled
	cpu.WriteSPSR(mode, cpsr)
	if cpu.IsThumb() {
		cpu.WriteReg(14, (pc-2)|1) // PC+2
	} else {
		cpu.WriteReg(14, pc-4)
	}
	cpu.CPSR &= ^BitT // force ARM state
	cpu.WriteReg(15, addr)
}

func (cpu *CPU) ResetPipeline() {
	cpu.ShouldResetPipeline = false
	pc := cpu.ReadReg(15)
	if cpu.IsThumb() {
		cpu.Pipeline[0] = uint32(cpu.Bus.Read16(pc + 2))
		cpu.Pipeline[1] = uint32(cpu.Bus.Read16(pc))
		cpu.reg[15] = pc + 4
	} else {
		cpu.Pipeline[0] = cpu.Bus.Read32(pc + 4)
		cpu.Pipeline[1] = cpu.Bus.Read32(pc)
		cpu.reg[15] = pc + 8
	}
}

func (cpu *CPU) AdvancePipeline() {
	pc := cpu.ReadReg(15)
	if cpu.IsThumb() {
		cpu.Pipeline[1] = cpu.Pipeline[0]
		cpu.Pipeline[0] = uint32(cpu.Bus.Read16(pc))
		cpu.reg[15] = pc + 2
	} else {
		cpu.Pipeline[1] = cpu.Pipeline[0]
		cpu.Pipeline[0] = cpu.Bus.Read32(pc)
		cpu.reg[15] = pc + 4
	}
}

func (cpu *CPU) Step() {
	if (cpu.CPSR&BitI) == 0 && (cpu.IRQ.IME&1) != 0 && (cpu.IRQ.IF&cpu.IRQ.IE) != 0 {
		cpu.HandleException(ExceptNormalInterrupt)
		cpu.ResetPipeline()
		return
	}

	opcode := cpu.Pipeline[1]

	if cpu.IsThumb() {
		cpu.ExecuteThumb(uint16(opcode))
	} else {
		cpu.ExecuteARM(opcode)
	}

	if cpu.ShouldResetPipeline {
		cpu.ResetPipeline()
	} else {
		cpu.AdvancePipeline()
	}
}

func IsBranchExchange(opcode uint32) bool {
	const branchExchangeFormat = 0b0000_0001_0010_1111_1111_1111_0001_0000
	const formatMask = 0b0000_1111_1111_1111_1111_1111_1111_0000
	return (opcode & formatMask) == branchExchangeFormat
}

func IsBlockDataTransfer(opcode uint32) bool {
	const blockDataTransferFormat = 0b0000_1000_0000_0000_0000_0000_0000_0000
	const formatMask = 0b0000_1110_0000_0000_0000_0000_0000_0000
	return (opcode & formatMask) == blockDataTransferFormat
}

func IsBranchAndBranchWithLink(opcode uint32) bool {
	const branchFormat = 0b0000_1010_0000_0000_0000_0000_0000_0000
	const formatMask = 0b0000_1110_0000_0000_0000_0000_0000_0000
	return (opcode & formatMask) == branchFormat
}

func IsSoftwareInterrupt(opcode uint32) bool {
	const softwareInterruptFormat = 0b0000_1111_0000_0000_0000_0000_0000_0000
	const formatMask = 0b0000_1111_0000_0000_0000_0000_0000_0000
	return (opcode & formatMask) == softwareInterruptFormat
}

func IsUndefined(opcode uint32) bool {
	const undefinedFormat = 0b0000_0110_0000_0000_0000_0000_0001_0000
	const formatMask = 0b0000_1110_0000_0000_0000_0000_0001_0000
	return (opcode & formatMask) == undefinedFormat
}

func IsSingleDataTransfer(opcode uint32) bool {
	const singleDataTransferFormat = 0b0000_0100_0000_0000_0000_0000_0000_0000
	const formatMask = 0b0000_1100_0000_0000_0000_0000_0000_0000
	return (opcode & formatMask) == singleDataTransferFormat
}

func IsSingleDataSwap(opcode uint32) bool {
	const singleDataSwapFormat = 0b0000_0001_0000_0000_0000_0000_1001_0000
	const formatMask = 0b0000_1111_1000_0000_0000_1111_1111_0000
	return (opcode & formatMask) == singleDataSwapFormat
}

func IsMultiply(opcode uint32) bool {
	const multiplyFormat = 0b0000_0000_0000_0000_0000_0000_1001_0000
	const formatMask = 0b0000_1111_1100_0000_0000_0000_1111_0000
	return (opcode & formatMask) == multiplyFormat
}

func IsMultiplyLong(opcode uint32) bool {
	const multiplyLongFormat = 0b0000_0000_1000_0000_0000_0000_1001_0000
	const formatMask = 0b0000_1111_1000_0000_0000_0000_1111_0000
	return (opcode & formatMask) == multiplyLongFormat
}

func IsHalfwordDataTransferRegister(opcode uint32) bool {
	const halfwordDataTransferRegisterFormat = 0b0000_0000_0000_0000_0000_0000_1001_0000
	const formatMask = 0b0000_1110_0100_0000_0000_1111_1001_0000
	return (opcode & formatMask) == halfwordDataTransferRegisterFormat
}

func IsHalfwordDataTransferImmediate(opcode uint32) bool {
	const halfwordDataTransferImmediateFormat = 0b0000_0000_0100_0000_0000_0000_1001_0000
	const formatMask = 0b0000_1110_0100_0000_0000_0000_1001_0000
	return (opcode & formatMask) == halfwordDataTransferImmediateFormat
}

func IsPSRTransferMRS(opcode uint32) bool {
	const mrsFormat = 0b0000_0001_0000_1111_0000_0000_0000_0000
	const formatMask = 0b0000_1111_1011_1111_0000_0000_0000_0000
	return (opcode & formatMask) == mrsFormat
}

func IsPSRTransferMSR(opcode uint32) bool {
	const msrFormat = 0b0000_0001_0010_0000_1111_0000_0000_0000
	const formatMask = 0b0000_1101_1011_0000_1111_0000_0000_0000
	return (opcode & formatMask) == msrFormat
}

func IsDataProcessing(opcode uint32) bool {
	const dataProcessingFormat = 0b0000_0000_0000_0000_0000_0000_0000_0000
	const formatMask = 0b0000_1100_0000_0000_0000_0000_0000_0000
	return (opcode & formatMask) == dataProcessingFormat
}

func Shift(value uint32, op int, amount uint, oldCarry bool, isRegisterShift bool) (result uint32, carry bool) {
	if isRegisterShift && amount == 0 {
		return value, oldCarry
	}
	switch op {
	case 0x0: // LSL
		if amount == 0 {
			result = value
			carry = oldCarry
		} else {
			amount &= 0xFF
			result = value << amount
			carry = (value>>(32-amount))&1 != 0
		}
	case 0x1: // LSR
		if amount == 0 {
			result = 0
			carry = (value>>31)&1 != 0
		} else {
			result = value >> amount
			carry = (value>>(amount-1))&1 != 0
		}
	case 0x2: // ASR
		if amount == 0 {
			if (value & 0x80000000) != 0 {
				result = 0xFFFFFFFF
				carry = true
			} else {
				result = 0
				carry = false
			}
		} else {
			result = uint32(int32(value) >> amount)
			carry = (value>>(amount-1))&1 != 0
		}
	case 0x3: // ROR
		if amount == 0 { // RXX
			carry = (value & 1) != 0
			result = value >> 1
			if oldCarry {
				result |= 1 << 31
			}
		} else {
			result, carry = ROR(value, amount%32)
		}
	}
	return
}

func ROR(val uint32, amount uint) (result uint32, carry bool) {
	if amount == 0 {
		return val, (val & (1 << 31)) != 0
	}
	result = (val >> amount) | (val << (32 - amount))
	carry = (val>>(amount-1))&1 != 0
	return
}

func (cpu *CPU) executeBranchExchange(opcode uint32) {
	Rn := int(opcode & 0xF)
	val := cpu.ReadReg(Rn)
	isThumb := (val & 1) != 0
	if isThumb {
		cpu.CPSR |= BitT
		cpu.WriteReg(15, val&0xFFFFFFFE)
	} else {
		cpu.CPSR &= ^BitT
		cpu.WriteReg(15, val)
	}
}

func (cpu *CPU) executeBlockDataTransfer(opcode uint32) {
	P := (opcode & (1 << 24)) != 0
	U := (opcode & (1 << 23)) != 0
	S := (opcode & (1 << 22)) != 0
	W := (opcode & (1 << 21)) != 0
	L := (opcode & (1 << 20)) != 0
	Rn := int((opcode >> 16) & 0xF)
	Rlist := opcode & 0xFFFF

	if Rlist == 0 {
		RnVal := cpu.ReadReg(Rn)
		var addr uint32
		if U {
			if P {
				addr = RnVal + 0x4
			} else {
				addr = RnVal
			}
		} else {
			if P {
				addr = RnVal - 0x40
			} else {
				addr = RnVal - 0x3C
			}
		}
		if L {
			val := cpu.Bus.Read32(addr)
			cpu.WriteReg(15, val)
		} else {
			cpu.Bus.Write32(addr, cpu.ReadReg(15)+4)
		}
		if W {
			if U {
				cpu.WriteReg(Rn, RnVal+0x40)
			} else {
				cpu.WriteReg(Rn, RnVal-0x40)
			}
		}
		return
	}

	firstReg := 0
	for (Rlist & (1 << firstReg)) == 0 {
		firstReg++
	}

	count := bits.OnesCount32(Rlist)
	RnVal := cpu.ReadReg(Rn)
	var addr uint32
	if U { // up
		if P { // pre
			addr = RnVal + 4
		} else {
			addr = RnVal
		}
	} else {
		if P {
			addr = RnVal - 4*uint32(count)
		} else {
			addr = RnVal - 4*(uint32(count)-1)
		}
	}

	if W && (L || firstReg != Rn) {
		if U {
			cpu.WriteReg(Rn, RnVal+4*uint32(count))
		} else {
			cpu.WriteReg(Rn, RnVal-4*uint32(count))
		}
	}

	for i := range 16 {
		if (Rlist & (1 << i)) != 0 {
			if L { // load
				val := cpu.Bus.Read32(addr)
				if S {
					cpu.WriteUserReg(i, val)
					if i == 15 {
						cpu.CPSR = cpu.ReadSPSR(cpu.Mode())
					}
				} else {
					cpu.WriteReg(i, val)
				}
			} else { // store
				var val uint32
				if S {
					val = cpu.ReadUserReg(i)
				} else {
					val = cpu.ReadReg(i)
				}
				if i == 15 {
					val += 4
				}
				cpu.Bus.Write32(addr, val)
			}
			addr += 4
		}
	}

	if W && (!L && firstReg == Rn) {
		if U {
			cpu.WriteReg(Rn, RnVal+4*uint32(count))
		} else {
			cpu.WriteReg(Rn, RnVal-4*uint32(count))
		}
	}
}

func (cpu *CPU) executeBranchAndBranchWithLink(opcode uint32) {
	pc := cpu.ReadReg(15) // PC+8
	offset := opcode & 0xFFFFFF
	// Sign Extension
	if (offset & 0x00800000) != 0 {
		offset |= 0xFF000000
	}
	offset <<= 2
	withLink := (opcode & (1 << 24)) != 0
	if withLink {
		cpu.WriteReg(14, pc-4) // PC+4
	}
	cpu.WriteReg(15, pc+uint32(int32(offset)))
}

func (cpu *CPU) executeSoftwareInterrupt() {
	cpu.HandleException(ExceptSoftwareInterrupt)
}

func (cpu *CPU) executeUndefined() {
	cpu.HandleException(ExceptUndefined)
}

func (cpu *CPU) executeSingleDataTransfer(opcode uint32) {
	I := (opcode & (1 << 25)) != 0
	P := (opcode & (1 << 24)) != 0
	U := (opcode & (1 << 23)) != 0
	B := (opcode & (1 << 22)) != 0
	W := (opcode & (1 << 21)) != 0
	L := (opcode & (1 << 20)) != 0
	Rn := int((opcode >> 16) & 0xF)
	Rd := int((opcode >> 12) & 0xF)

	var offset uint32
	if I { // shifted register
		shiftAmount := uint((opcode >> 7) & 0x1F)
		shiftType := int((opcode >> 5) & 0x3)
		Rm := int(opcode & 0xF)
		RmVal := cpu.ReadReg(Rm)
		offset, _ = Shift(RmVal, shiftType, shiftAmount, (cpu.CPSR&BitC) != 0, false)
	} else { // immediate
		offset = opcode & 0xFFF
	}

	RnVal := cpu.ReadReg(Rn)
	addr := RnVal
	if P { // pre
		if U { // up
			addr += offset
		} else { // down
			addr -= offset
		}
	}
	RdVal := cpu.ReadReg(Rd)
	if Rd == 15 {
		RdVal += 4
	}

	if W || !P {
		if U {
			cpu.WriteReg(Rn, RnVal+offset)
		} else {
			cpu.WriteReg(Rn, RnVal-offset)
		}
	}

	if L { // load
		if B {
			val := cpu.Bus.Read8(addr)
			cpu.WriteReg(Rd, uint32(val))
		} else {
			val := cpu.Bus.Read32(addr & 0xFFFFFFFC)
			val, _ = ROR(val, uint(addr&0x3)*8)
			cpu.WriteReg(Rd, val)
		}
	} else { // store
		if B {
			cpu.Bus.Write8(addr, byte(RdVal&0xFF))
		} else {
			cpu.Bus.Write32(addr&0xFFFFFFFC, RdVal)
		}
	}
}

func (cpu *CPU) executeSingleDataSwap(opcode uint32) {
	B := (opcode & (1 << 22)) != 0
	Rn := int((opcode >> 16) & 0xF)
	Rd := int((opcode >> 12) & 0xF)
	Rm := int(opcode & 0xF)
	addr := cpu.ReadReg(Rn)
	RmVal := cpu.ReadReg(Rm)
	var memVal uint32
	if B {
		memVal = uint32(cpu.Bus.Read8(addr))
		cpu.Bus.Write8(addr, byte(RmVal&0xFF))
	} else {
		memVal = cpu.Bus.Read32(addr & 0xFFFFFFFC)
		memVal, _ = ROR(memVal, uint(addr&0x3)*8)
		cpu.Bus.Write32(addr&0xFFFFFFFC, RmVal)
	}
	cpu.WriteReg(Rd, memVal)
}

func (cpu *CPU) executeMultiply(opcode uint32) {
	op := (opcode >> 21) & 0xF
	S := (opcode & (1 << 20)) != 0
	Rd := int((opcode >> 16) & 0xF) // or RdHi
	Rn := int((opcode >> 12) & 0xF) // or RdLo
	Rs := int((opcode >> 8) & 0xF)
	Rm := int(opcode & 0xF)

	var flags uint32
	var flagsMask uint32 = BitN | BitZ

	switch op {
	case 0x0: // MUL
		RmVal := cpu.ReadReg(Rm)
		RsVal := cpu.ReadReg(Rs)
		result := RmVal * RsVal
		cpu.WriteReg(Rd, result)
		if (result & (1 << 31)) != 0 {
			flags |= BitN
		}
		if result == 0 {
			flags |= BitZ
		}
	case 0x1: // MLA
		RmVal := cpu.ReadReg(Rm)
		RsVal := cpu.ReadReg(Rs)
		RnVal := cpu.ReadReg(Rn)
		result := RmVal*RsVal + RnVal
		cpu.WriteReg(Rd, result)
		if (result & (1 << 31)) != 0 {
			flags |= BitN
		}
		if result == 0 {
			flags |= BitZ
		}
	case 0x4: // UMULL
		RmVal := cpu.ReadReg(Rm)
		RsVal := cpu.ReadReg(Rs)
		result := uint64(RmVal) * uint64(RsVal)
		hi := uint32(result >> 32)
		lo := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, hi)
		cpu.WriteReg(Rn, lo)
		if (hi & (1 << 31)) != 0 {
			flags |= BitN
		}
		if result == 0 {
			flags |= BitZ
		}
	case 0x5: // UMLAL
		RdVal := cpu.ReadReg(Rd)
		RnVal := cpu.ReadReg(Rn)
		acc := uint64(RdVal)<<32 | uint64(RnVal)
		RmVal := cpu.ReadReg(Rm)
		RsVal := cpu.ReadReg(Rs)
		result := uint64(RmVal)*uint64(RsVal) + acc
		hi := uint32(result >> 32)
		lo := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, hi)
		cpu.WriteReg(Rn, lo)
		if (hi & (1 << 31)) != 0 {
			flags |= BitN
		}
		if result == 0 {
			flags |= BitZ
		}
	case 0x6: // SMULL
		RmVal := cpu.ReadReg(Rm)
		RsVal := cpu.ReadReg(Rs)
		result := int64(int32(RmVal)) * int64(int32(RsVal))
		hi := uint32(result >> 32)
		lo := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, hi)
		cpu.WriteReg(Rn, lo)
		if (hi & (1 << 31)) != 0 {
			flags |= BitN
		}
		if result == 0 {
			flags |= BitZ
		}
	case 0x7: // SMLAL
		RdVal := cpu.ReadReg(Rd)
		RnVal := cpu.ReadReg(Rn)
		acc := (uint64(RdVal) << 32) | uint64(RnVal)
		RmVal := cpu.ReadReg(Rm)
		RsVal := cpu.ReadReg(Rs)
		result := uint64(int64(int32(RmVal))*int64(int32(RsVal))) + acc
		hi := uint32(result >> 32)
		lo := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, hi)
		cpu.WriteReg(Rn, lo)
		if (hi & (1 << 31)) != 0 {
			flags |= BitN
		}
		if result == 0 {
			flags |= BitZ
		}
	}

	if S {
		cpu.CPSR = (cpu.CPSR & ^flagsMask) | (flags & flagsMask)
	}
}

func (cpu *CPU) executeHalfwordDataTransfer(opcode uint32) {
	P := (opcode & (1 << 24)) != 0
	U := (opcode & (1 << 23)) != 0
	I := (opcode & (1 << 22)) != 0
	W := (opcode & (1 << 21)) != 0
	L := (opcode & (1 << 20)) != 0
	Rn := int((opcode >> 16) & 0xF)
	Rd := int((opcode >> 12) & 0xF)
	op := (opcode >> 5) & 0x3

	var offset uint32
	if I { // immediate
		offset = (((opcode >> 8) & 0xF) << 4) | (opcode & 0xF)
	} else { // register
		Rm := int(opcode & 0xF)
		offset = cpu.ReadReg(Rm)
	}

	RnVal := cpu.ReadReg(Rn)
	addr := RnVal
	if P { // pre
		if U { // up
			addr += offset
		} else { // down
			addr -= offset
		}
	}
	RdVal := cpu.ReadReg(Rd)
	if Rd == 15 {
		RdVal += 4
	}

	if W || !P {
		if U {
			cpu.WriteReg(Rn, RnVal+offset)
		} else {
			cpu.WriteReg(Rn, RnVal-offset)
		}
	}

	if L { // load
		var val uint32
		switch op {
		case 0x1: // LDRH
			val = uint32(cpu.Bus.Read16(addr & 0xFFFFFFFE))
			if (addr & 0x1) != 0 {
				val, _ = ROR(val, 8)
			}
		case 0x2: // LDRSB
			val = uint32(cpu.Bus.Read8(addr))
			if (val & 0x80) != 0 {
				val |= 0xFFFFFF00
			}
		case 0x3: // LDRSH
			val = uint32(cpu.Bus.Read16(addr & 0xFFFFFFFE))
			if (addr & 0x1) != 0 {
				val >>= 8
				if (val & 0x80) != 0 {
					val |= 0xFFFFFF00
				}
			} else {
				if (val & 0x8000) != 0 {
					val |= 0xFFFF0000
				}
			}
		}
		cpu.WriteReg(Rd, val)
	} else { // store
		switch op {
		case 0x1: // STRH
			cpu.Bus.Write16(addr&0xFFFFFFFE, uint16(RdVal&0xFFFF))
		case 0x2: // LDRD
			cpu.WriteReg(Rd, cpu.Bus.Read32(addr))
			cpu.WriteReg(Rd+1, cpu.Bus.Read32(addr+4))
		case 0x3: // STRD
			cpu.Bus.Write32(addr, cpu.ReadReg(Rd))
			cpu.Bus.Write32(addr+4, cpu.ReadReg(Rd+1))
		}
	}
}

func (cpu *CPU) executePSRTransferMRS(opcode uint32) {
	src := (opcode & (1 << 22)) != 0
	Rd := int((opcode >> 12) & 0xF)
	var psr uint32
	if src {
		mode := cpu.Mode()
		psr = cpu.ReadSPSR(mode)
	} else {
		psr = cpu.CPSR
	}
	cpu.WriteReg(Rd, psr)
}

func (cpu *CPU) executePSRTransferMSR(opcode uint32) {
	I := (opcode & (1 << 25)) != 0
	dst := (opcode & (1 << 22)) != 0
	var mask uint32
	if (opcode & (1 << 19)) != 0 {
		mask |= 0xFF000000 // flags field
	}
	if (opcode & (1 << 18)) != 0 {
		mask |= 0x00FF0000 // status field
	}
	if (opcode & (1 << 17)) != 0 {
		mask |= 0x0000FF00 // extension field
	}
	if (opcode & (1 << 16)) != 0 {
		mask |= 0x000000FF // control field
	}
	var val uint32
	if I { // immediate
		shiftAmount := uint((opcode>>8)&0xF) * 2
		imm := opcode & 0xFF
		val, _ = ROR(imm, shiftAmount)
	} else { // register
		Rm := int(opcode & 0xF)
		val = cpu.ReadReg(Rm)
	}
	if dst {
		mode := cpu.Mode()
		spsr := cpu.ReadSPSR(mode)
		cpu.WriteSPSR(mode, (spsr & ^mask)|(val&mask))
	} else {
		cpsr := cpu.CPSR
		cpu.CPSR = (cpsr & ^mask) | (val & mask)
	}
}

func (cpu *CPU) executeDataProcessing(opcode uint32) {
	I := (opcode & (1 << 25)) != 0
	op := (opcode >> 21) & 0xF
	S := (opcode & (1 << 20)) != 0
	Rn := int((opcode >> 16) & 0xF)
	Rd := int((opcode >> 12) & 0xF)

	operand1 := cpu.ReadReg(Rn)
	var operand2 uint32
	var shiftCarry bool
	if I { // immediate
		shiftAmount := uint((opcode>>8)&0xF) << 1
		imm := opcode & 0xFF
		operand2, shiftCarry = ROR(imm, shiftAmount)
	} else { // register
		shiftType := int((opcode >> 5) & 0x3)
		R := (opcode & (1 << 4)) != 0
		Rm := int(opcode & 0xF)
		RmVal := cpu.ReadReg(Rm)
		if R {
			if Rn == 15 {
				operand1 += 4
			}
			if Rm == 15 {
				RmVal += 4
			}
		}
		var shiftAmount uint
		if R { // register
			Rs := int((opcode >> 8) & 0xF)
			shiftAmount = uint(cpu.ReadReg(Rs))
		} else { // immediate
			shiftAmount = uint((opcode >> 7) & 0x1F)
		}
		operand2, shiftCarry = Shift(RmVal, shiftType, shiftAmount, (cpu.CPSR&BitC) != 0, R)
	}

	switch op {
	case 0x0: // AND
		result := operand1 & operand2
		cpu.WriteReg(Rd, result)
		if S {
			cpu.UpdateLogicalFlags(result, shiftCarry)
		}
	case 0x1: // EOR
		result := operand1 ^ operand2
		cpu.WriteReg(Rd, result)
		if S {
			cpu.UpdateLogicalFlags(result, shiftCarry)
		}
	case 0x2: // SUB
		result := operand1 - operand2
		cpu.WriteReg(Rd, result)
		if S {
			carry := operand1 >= operand2
			overflow := CalculateOverflow(operand1, operand2, result, true)
			cpu.UpdateArithmeticFlags(result, carry, overflow)
		}
	case 0x3: // RSB
		result := operand2 - operand1
		cpu.WriteReg(Rd, result)
		if S {
			carry := operand2 >= operand1
			overflow := CalculateOverflow(operand2, operand1, result, true)
			cpu.UpdateArithmeticFlags(result, carry, overflow)
		}
	case 0x4: // ADD
		result := uint64(operand1) + uint64(operand2)
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		if S {
			carry := result > 0xFFFFFFFF
			overflow := CalculateOverflow(operand1, operand2, finalResult, false)
			cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
		}
	case 0x5: // ADC
		result := uint64(operand1) + uint64(operand2)
		if (cpu.CPSR & BitC) != 0 {
			result += 1
		}
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		if S {
			carry := result > 0xFFFFFFFF
			overflow := CalculateOverflow(operand1, operand2, finalResult, false)
			cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
		}
	case 0x6: // SBC
		var carryIn uint64 = 0
		if (cpu.CPSR & BitC) == 0 {
			carryIn = 1
		}
		result := uint64(operand1) - uint64(operand2) - carryIn
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		if S {
			carry := uint64(operand1) >= uint64(operand2)+carryIn
			overflow := CalculateOverflow(operand1, operand2, finalResult, true)
			cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
		}
	case 0x7: // RSC
		var carryIn uint64 = 0
		if (cpu.CPSR & BitC) == 0 {
			carryIn = 1
		}
		result := uint64(operand2) - uint64(operand1) - carryIn
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		if S {
			carry := uint64(operand2) >= uint64(operand1)+carryIn
			overflow := CalculateOverflow(operand2, operand1, finalResult, true)
			cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
		}
	case 0x8: // TST
		result := operand1 & operand2
		cpu.UpdateLogicalFlags(result, shiftCarry)
	case 0x9: // TEQ
		result := operand1 ^ operand2
		cpu.UpdateLogicalFlags(result, shiftCarry)
	case 0xA: // CMP
		result := operand1 - operand2
		carry := operand1 >= operand2
		overflow := CalculateOverflow(operand1, operand2, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	case 0xB: // CMN
		result := uint64(operand1) + uint64(operand2)
		finalResult := uint32(result & 0xFFFFFFFF)
		carry := result > 0xFFFFFFFF
		overflow := CalculateOverflow(operand1, operand2, finalResult, false)
		cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
	case 0xC: // ORR
		result := operand1 | operand2
		cpu.WriteReg(Rd, result)
		if S {
			cpu.UpdateLogicalFlags(result, shiftCarry)
		}
	case 0xD: // MOV
		result := operand2
		cpu.WriteReg(Rd, result)
		if S {
			cpu.UpdateLogicalFlags(result, shiftCarry)
		}
	case 0xE: // BIC
		result := operand1 & ^operand2
		cpu.WriteReg(Rd, result)
		if S {
			cpu.UpdateLogicalFlags(result, shiftCarry)
		}
	case 0xF: // MVN
		result := ^operand2
		cpu.WriteReg(Rd, result)
		if S {
			cpu.UpdateLogicalFlags(result, shiftCarry)
		}
	}

	if S && Rd == 15 {
		cpu.CPSR = cpu.ReadSPSR(cpu.Mode())
	}
}

func (cpu *CPU) checkCondition(opcode uint32) bool {
	cond := (opcode >> 28) & 0xF
	flagN, flagZ, flagC, flagV := cpu.GetFlags()

	switch cond {
	case 0x0: // EQ
		return flagZ
	case 0x1: // NE
		return !flagZ
	case 0x2: // CS
		return flagC
	case 0x3: // CC
		return !flagC
	case 0x4: // MI
		return flagN
	case 0x5: // PL
		return !flagN
	case 0x6: // VS
		return flagV
	case 0x7: // VC
		return !flagV
	case 0x8: // HI
		return flagC && !flagZ
	case 0x9: // LS
		return !flagC || flagZ
	case 0xA: // GE
		return flagN == flagV
	case 0xB: // LT
		return flagN != flagV
	case 0xC: // GT
		return !flagZ && flagN == flagV
	case 0xD: // LE
		return flagZ || flagN != flagV
	case 0xE: // AL
		return true
	}
	// Never
	return false
}

func (cpu *CPU) ExecuteARM(opcode uint32) {
	if !cpu.checkCondition(opcode) {
		return
	}
	switch {
	case IsBranchExchange(opcode):
		cpu.executeBranchExchange(opcode)
	case IsBlockDataTransfer(opcode):
		cpu.executeBlockDataTransfer(opcode)
	case IsBranchAndBranchWithLink(opcode):
		cpu.executeBranchAndBranchWithLink(opcode)
	case IsSoftwareInterrupt(opcode):
		cpu.executeSoftwareInterrupt()
	case IsUndefined(opcode):
		cpu.executeUndefined()
	case IsSingleDataTransfer(opcode):
		cpu.executeSingleDataTransfer(opcode)
	case IsSingleDataSwap(opcode):
		cpu.executeSingleDataSwap(opcode)
	case IsMultiply(opcode) || IsMultiplyLong(opcode):
		cpu.executeMultiply(opcode)
	case IsHalfwordDataTransferRegister(opcode):
		cpu.executeHalfwordDataTransfer(opcode)
	case IsHalfwordDataTransferImmediate(opcode):
		cpu.executeHalfwordDataTransfer(opcode)
	case IsPSRTransferMRS(opcode):
		cpu.executePSRTransferMRS(opcode)
	case IsPSRTransferMSR(opcode):
		cpu.executePSRTransferMSR(opcode)
	case IsDataProcessing(opcode):
		cpu.executeDataProcessing(opcode)
	}
}

func IsThumbSoftwareInterrupt(opcode uint16) bool {
	const softwareInterruptFormat = 0b1101_1111_0000_0000
	const formatMask = 0b1111_1111_0000_0000
	return (opcode & formatMask) == softwareInterruptFormat
}

func IsThumbUnconditionalBranch(opcode uint16) bool {
	const unconditionalBranchFormat = 0b1110_0000_0000_0000
	const formatMask = 0b1111_1000_0000_0000
	return (opcode & formatMask) == unconditionalBranchFormat
}

func IsThumbConditionalBranch(opcode uint16) bool {
	const conditionalBranchFormat = 0b1101_0000_0000_0000
	const formatMask = 0b1111_0000_0000_0000
	return (opcode & formatMask) == conditionalBranchFormat
}

func IsThumbMultipleLoadStore(opcode uint16) bool {
	const multipleLoadStoreFormat = 0b1100_0000_0000_0000
	const formatMask = 0b1111_0000_0000_0000
	return (opcode & formatMask) == multipleLoadStoreFormat
}

func IsThumbLongBranchWithLink(opcode uint16) bool {
	const longBranchWithLinkFormat = 0b1111_0000_0000_0000
	const formatMask = 0b1111_0000_0000_0000
	return (opcode & formatMask) == longBranchWithLinkFormat
}

func IsThumbAddOffsetToStackPointer(opcode uint16) bool {
	const addOffsetToStackPointerFormat = 0b1011_0000_0000_0000
	const formatMask = 0b1111_1111_0000_0000
	return (opcode & formatMask) == addOffsetToStackPointerFormat
}

func IsThumbPushPopRegisters(opcode uint16) bool {
	const pushPopRegistersFormat = 0b1011_0100_0000_0000
	const formatMask = 0b1111_0110_0000_0000
	return (opcode & formatMask) == pushPopRegistersFormat
}

func IsThumbLoadStoreHalfword(opcode uint16) bool {
	const loadStoreHalfwordFormat = 0b1000_0000_0000_0000
	const formatMask = 0b1111_0000_0000_0000
	return (opcode & formatMask) == loadStoreHalfwordFormat
}

func IsThumbSPRelativeLoadStore(opcode uint16) bool {
	const spRelativeLoadStoreFormat = 0b1001_0000_0000_0000
	const formatMask = 0b1111_0000_0000_0000
	return (opcode & formatMask) == spRelativeLoadStoreFormat
}

func IsThumbLoadAddress(opcode uint16) bool {
	const loadAddressFormat = 0b1010_0000_0000_0000
	const formatMask = 0b1111_0000_0000_0000
	return (opcode & formatMask) == loadAddressFormat
}

func IsThumbLoadStoreWithImmediateOffset(opcode uint16) bool {
	const loadStoreImmediateOffsetFormat = 0b0110_0000_0000_0000
	const formatMask = 0b1110_0000_0000_0000
	return (opcode & formatMask) == loadStoreImmediateOffsetFormat
}

func IsThumbLoadStoreWithRegisterOffset(opcode uint16) bool {
	const loadStoreWithRegisterOffsetFormat = 0b0101_0000_0000_0000
	const formatMask = 0b1111_0010_0000_0000
	return (opcode & formatMask) == loadStoreWithRegisterOffsetFormat
}

func IsThumbLoadStoreSignExtendedByteHalfword(opcode uint16) bool {
	const loadStoreSignExtendedByteHalfwordFormat = 0b0101_0010_0000_0000
	const formatMask = 0b1111_0010_0000_0000
	return (opcode & formatMask) == loadStoreSignExtendedByteHalfwordFormat
}

func IsThumbPCRelativeLoad(opcode uint16) bool {
	const pcRelativeLoadFormat = 0b0100_1000_0000_0000
	const formatMask = 0b1111_1000_0000_0000
	return (opcode & formatMask) == pcRelativeLoadFormat
}

func IsThumbHiRegisterOperationsBranchExchange(opcode uint16) bool {
	const hiRegisterOperationsBranchExchangeFormat = 0b0100_0100_0000_0000
	const formatMask = 0b1111_1100_0000_0000
	return (opcode & formatMask) == hiRegisterOperationsBranchExchangeFormat
}

func IsThumbALUOperations(opcode uint16) bool {
	const aluOperationsFormat = 0b0100_0000_0000_0000
	const formatMask = 0b1111_1100_0000_0000
	return (opcode & formatMask) == aluOperationsFormat
}

func IsThumbMoveCompareAddSubtractImmediate(opcode uint16) bool {
	const moveCompareAddSubtractImmediateFormat = 0b0010_0000_0000_0000
	const formatMask = 0b1110_0000_0000_0000
	return (opcode & formatMask) == moveCompareAddSubtractImmediateFormat
}

func IsThumbAddSubtract(opcode uint16) bool {
	const addSubtractFormat = 0b0001_1000_0000_0000
	const formatMask = 0b1111_1000_0000_0000
	return (opcode & formatMask) == addSubtractFormat
}

func IsThumbMoveShiftedRegister(opcode uint16) bool {
	const moveShiftedRegistersFormat = 0b0000_0000_0000_0000
	const formatMask = 0b1110_0000_0000_0000
	return (opcode & formatMask) == moveShiftedRegistersFormat

}

func (cpu *CPU) executeThumbSoftwareInterrupt() {
	cpu.HandleException(ExceptSoftwareInterrupt)
}

func (cpu *CPU) executeThumbUnconditionalBranch(opcode uint16) {
	offset := uint32(opcode & 0x7FF)
	if (offset & 0x400) != 0 {
		offset |= 0xFFFFF800
	}
	offset <<= 1
	addr := cpu.ReadReg(15) + offset

	cpu.WriteReg(15, addr)
}

func (cpu *CPU) executeThumbConditionalBranch(opcode uint16) {
	cond := (opcode >> 8) & 0xF
	offset := uint32(int32(int8(opcode&0xFF))) << 1
	flagN, flagZ, flagC, flagV := cpu.GetFlags()

	branch := false

	switch cond {
	case 0x0: // BEQ
		branch = flagZ
	case 0x1: // BNE
		branch = !flagZ
	case 0x2: // BCS
		branch = flagC
	case 0x3: // BCC
		branch = !flagC
	case 0x4: // BMI
		branch = flagN
	case 0x5: // BPL
		branch = !flagN
	case 0x6: // BVS
		branch = flagV
	case 0x7: // BVC
		branch = !flagV
	case 0x8: // BHI
		branch = flagC && !flagZ
	case 0x9: // BLS
		branch = !flagC || flagZ
	case 0xA: // BGE
		branch = flagN == flagV
	case 0xB: // BLT
		branch = flagN != flagV
	case 0xC: // BGT
		branch = !flagZ && flagN == flagV
	case 0xD: // BLE
		branch = flagZ || flagN != flagV
	}

	if branch {
		cpu.WriteReg(15, cpu.ReadReg(15)+offset)
	}
}

func (cpu *CPU) executeThumbMultipleLoadStore(opcode uint16) {
	op := int((opcode >> 11) & 1)
	Rb := int((opcode >> 8) & 0x7)
	Rlist := opcode & 0xFF

	addr := cpu.ReadReg(Rb)

	count := bits.OnesCount16(Rlist)

	if Rlist == 0 {
		if op == 0 {
			cpu.Bus.Write32(addr, cpu.ReadReg(15)+2)
		} else {
			cpu.WriteReg(15, cpu.Bus.Read32(addr))
		}
		RbVal := cpu.ReadReg(Rb)
		cpu.WriteReg(Rb, RbVal+0x40)
		return
	}

	firstReg := 0
	for (Rlist & (1 << firstReg)) == 0 {
		firstReg++
	}

	if op == 0 && firstReg != Rb {
		cpu.WriteReg(Rb, addr+uint32(count)*4)
	}

	for i := range 8 {
		if (Rlist & (1 << i)) != 0 {
			switch op {
			case 0x0: // STM
				cpu.Bus.Write32(addr, cpu.ReadReg(i))
			case 0x1: // LDM
				cpu.WriteReg(i, cpu.Bus.Read32(addr))
			}
			addr += 4
		}
	}

	if op == 1 || firstReg == Rb {
		cpu.WriteReg(Rb, addr)
	}
}

func (cpu *CPU) executeThumbLongBranchWithLink(opcode uint16) {
	H := (opcode >> 11) & 1
	offset := uint32(opcode) & 0x7FF

	if H == 0 {
		// First instruction: BL setup (high part)
		if (offset & 0x400) != 0 {
			offset |= 0xFFFFF800
		}
		cpu.WriteReg(14, cpu.ReadReg(15)+(offset<<12))
	} else {
		// Second instruction: BL execute (low part)
		newPC := (cpu.ReadReg(14) + (offset << 1)) & 0xFFFFFFFE
		retAddr := (cpu.ReadReg(15) - 2) | 1
		cpu.WriteReg(15, newPC)
		cpu.WriteReg(14, retAddr)
	}
}

func (cpu *CPU) executeThumbAddOffsetToStackPointer(opcode uint16) {
	op := int((opcode >> 7) & 1)
	nn := uint32(opcode&0x7F) << 2

	switch op {
	case 0x0: // ADD SP,#nn
		cpu.WriteReg(13, cpu.ReadReg(13)+nn)
	case 0x1: // ADD SP,#-nn
		cpu.WriteReg(13, cpu.ReadReg(13)-nn)
	}
}

func (cpu *CPU) executeThumbPushPopRegisters(opcode uint16) {
	op := int((opcode >> 11) & 1)
	PC_LR := (opcode & (1 << 8)) != 0
	Rlist := opcode & 0xFF

	sp := cpu.ReadReg(13)

	switch op {
	case 0x0: // PUSH
		// PUSH LR
		if PC_LR {
			sp -= 4
			cpu.Bus.Write32(sp, cpu.ReadReg(14))
		}
		for i := 7; i >= 0; i-- {
			if (Rlist & (1 << i)) != 0 {
				sp -= 4
				cpu.Bus.Write32(sp, cpu.ReadReg(i))
			}
		}
	case 0x1: // POP
		for i := 0; i < 8; i++ {
			if (Rlist & (1 << i)) != 0 {
				cpu.WriteReg(i, cpu.Bus.Read32(sp))
				sp += 4
			}
		}
		// POP PC
		if PC_LR {
			cpu.WriteReg(15, cpu.Bus.Read32(sp)&0xFFFFFFFE)
			sp += 4
		}
	}

	cpu.WriteReg(13, sp)
}

func (cpu *CPU) executeThumbLoadStoreHalfword(opcode uint16) {
	op := int((opcode >> 11) & 1)
	nn := uint32((opcode>>6)&0x1F) << 1
	Rb := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)
	addr := cpu.ReadReg(Rb) + nn

	switch op {
	case 0x0: // STRH
		cpu.Bus.Write16(addr&0xFFFFFFFE, uint16(cpu.ReadReg(Rd)&0xFFFF))
	case 0x1: // LDRH
		val := uint32(cpu.Bus.Read16(addr & 0xFFFFFFFE))
		if (addr & 0x1) != 0 {
			val, _ = ROR(val, 8)
		}
		cpu.WriteReg(Rd, val)
	}
}

func (cpu *CPU) executeThumbSPRelativeLoadStore(opcode uint16) {
	op := int((opcode >> 11) & 1)
	Rd := int((opcode >> 8) & 0x7)
	nn := uint32(opcode&0xFF) << 2
	addr := cpu.ReadReg(13) + nn

	switch op {
	case 0x0: // STR
		cpu.Bus.Write32(addr&0xFFFFFFFC, cpu.ReadReg(Rd))
	case 0x1: // LDR
		val := cpu.Bus.Read32(addr & 0xFFFFFFFC)
		val, _ = ROR(val, uint(addr&0x3)*8)
		cpu.WriteReg(Rd, val)
	}
}

func (cpu *CPU) executeThumbLoadAddress(opcode uint16) {
	op := int((opcode >> 11) & 1)
	Rd := int((opcode >> 8) & 0x7)
	nn := uint32(opcode&0xFF) << 2

	switch op {
	case 0x0: // PC relative
		cpu.WriteReg(Rd, (cpu.ReadReg(15)&0xFFFFFFFD)+nn)
	case 0x1: // SP relative
		cpu.WriteReg(Rd, cpu.ReadReg(13)+nn)
	}
}

func (cpu *CPU) executeThumbLoadStoreWithImmediateOffset(opcode uint16) {
	op := int((opcode >> 11) & 0x3)
	nn := (uint32(opcode) >> 6) & 0x1F
	Rb := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)
	RbVal := cpu.ReadReg(Rb)

	switch op {
	case 0x0: // STR
		addr := RbVal + (nn << 2)
		cpu.Bus.Write32(addr&0xFFFFFFFC, cpu.ReadReg(Rd))
	case 0x1: // LDR
		addr := RbVal + (nn << 2)
		val := cpu.Bus.Read32(addr & 0xFFFFFFFC)
		val, _ = ROR(val, uint(addr&0x3)*8)
		cpu.WriteReg(Rd, val)
	case 0x2: // STRB
		addr := RbVal + nn
		cpu.Bus.Write8(addr, byte(cpu.ReadReg(Rd)&0xFF))
	case 0x3: // LDRB
		addr := RbVal + nn
		cpu.WriteReg(Rd, uint32(cpu.Bus.Read8(addr)))
	}
}

func (cpu *CPU) executeThumbLoadStoreWithRegisterOffset(opcode uint16) {
	op := int((opcode >> 10) & 0x3)
	Ro := int((opcode >> 6) & 0x7)
	Rb := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)

	addr := cpu.ReadReg(Rb) + cpu.ReadReg(Ro)

	switch op {
	case 0x0: // STR
		cpu.Bus.Write32(addr&0xFFFFFFFC, cpu.ReadReg(Rd))
	case 0x1: // STRB
		cpu.Bus.Write8(addr, byte(cpu.ReadReg(Rd)&0xFF))
	case 0x2: // LDR
		val := cpu.Bus.Read32(addr & 0xFFFFFFFC)
		val, _ = ROR(val, uint(addr&0x3)*8)
		cpu.WriteReg(Rd, val)
	case 0x3: // LDRB
		cpu.WriteReg(Rd, uint32(cpu.Bus.Read8(addr)))
	}
}

func (cpu *CPU) executeThumbLoadStoreSignExtendedByteHalfword(opcode uint16) {
	op := int((opcode >> 10) & 0x3)
	Ro := int((opcode >> 6) & 0x7)
	Rb := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)

	addr := cpu.ReadReg(Rb) + cpu.ReadReg(Ro)

	switch op {
	case 0x0: // STRH
		cpu.Bus.Write16(addr&0xFFFFFFFE, uint16(cpu.ReadReg(Rd)&0xFFFF))
	case 0x1: // LDRSB
		val := int32(int8(cpu.Bus.Read8(addr)))
		cpu.WriteReg(Rd, uint32(val))
	case 0x2: // LDRH
		val := uint32(cpu.Bus.Read16(addr & 0xFFFFFFFE))
		if (addr & 0x1) != 0 {
			val, _ = ROR(val, 8)
		}
		cpu.WriteReg(Rd, val)
	case 0x3: // LDRSH
		val := uint32(cpu.Bus.Read16(addr & 0xFFFFFFFE))
		if (addr & 0x1) != 0 {
			val >>= 8
			if (val & 0x80) != 0 {
				val |= 0xFFFFFF00
			}
		} else {
			if (val & 0x8000) != 0 {
				val |= 0xFFFF0000
			}
		}
		cpu.WriteReg(Rd, val)
	}
}

func (cpu *CPU) executeThumbPCRelativeLoad(opcode uint16) {
	Rd := int((opcode >> 8) & 0x7)
	nn := (uint32(opcode) & 0xFF) << 2

	pc := cpu.ReadReg(15) & 0xFFFFFFFC
	val := cpu.Bus.Read32(pc + nn)
	cpu.WriteReg(Rd, val)
}

func (cpu *CPU) executeThumbHiRegisterOperationsBranchExchange(opcode uint16) {
	op := int((opcode >> 8) & 0x3)
	MSBd := (opcode & (1 << 7)) != 0
	MSBs := (opcode & (1 << 6)) != 0
	Rs := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)
	if MSBd {
		Rd |= 0x8
	}
	if MSBs {
		Rs |= 0x8
	}
	RsVal := cpu.ReadReg(Rs)
	RdVal := cpu.ReadReg(Rd)

	switch op {
	case 0x0: // ADD
		result := RdVal + RsVal
		if Rd == 15 {
			cpu.WriteReg(15, result&0xFFFFFFFE)
		} else {
			cpu.WriteReg(Rd, result)
		}
	case 0x1: // CMP
		result := RdVal - RsVal
		var flags uint32
		if (result & (1 << 31)) != 0 {
			flags |= BitN
		}
		if result == 0 {
			flags |= BitZ
		}
		if RdVal >= RsVal {
			flags |= BitC
		}
		carry := RdVal >= RsVal
		overflow := CalculateOverflow(RdVal, RsVal, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	case 0x2: // MOV
		if Rd == 15 {
			cpu.WriteReg(15, RsVal&0xFFFFFFFE)
		} else {
			cpu.WriteReg(Rd, RsVal)
		}
	case 0x3: // BX
		if Rs == 15 {
			cpu.CPSR &= ^BitT
			cpu.WriteReg(15, RsVal&0xFFFFFFFC)
		} else {
			if RsVal&1 != 0 {
				cpu.WriteReg(15, RsVal&0xFFFFFFFE)
			} else {
				cpu.CPSR &= ^BitT
				cpu.WriteReg(15, RsVal)
			}
		}
	}
}

func (cpu *CPU) executeThumbALUOperations(opcode uint16) {
	op := int((opcode >> 6) & 0xF)
	Rs := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)
	RsVal := cpu.ReadReg(Rs)
	RdVal := cpu.ReadReg(Rd)

	switch op {
	case 0x0: // AND
		result := RdVal & RsVal
		cpu.WriteReg(Rd, result)
		cpu.UpdateLogicalFlags(result, false)
	case 0x1: // EOR
		result := RdVal ^ RsVal
		cpu.WriteReg(Rd, result)
		cpu.UpdateLogicalFlags(result, false)
	case 0x2: // LSL
		result := RdVal << (RsVal & 0xFF)
		cpu.WriteReg(Rd, result)
		carry := (cpu.CPSR & BitC) != 0
		if RsVal > 0 {
			carry = (RdVal & (1 << (32 - RsVal))) != 0
		}
		cpu.UpdateLogicalFlags(result, carry)
	case 0x3: // LSR
		result := RdVal >> (RsVal & 0xFF)
		cpu.WriteReg(Rd, result)
		carry := (cpu.CPSR & BitC) != 0
		if RsVal > 0 {
			carry = (RdVal & (1 << (RsVal - 1))) != 0
		}
		cpu.UpdateLogicalFlags(result, carry)
	case 0x4: // ASR
		RsVal &= 0xFF
		result := uint32(int32(RdVal) >> RsVal)
		cpu.WriteReg(Rd, result)
		carry := (cpu.CPSR & BitC) != 0
		if RsVal > 0 {
			carry = (RdVal & (1 << (RsVal - 1))) != 0
		}
		cpu.UpdateLogicalFlags(result, carry)
	case 0x5: // ADC
		result := uint64(RdVal) + uint64(RsVal)
		if (cpu.CPSR & BitC) != 0 {
			result += 1
		}
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		carry := result > 0xFFFFFFFF
		overflow := CalculateOverflow(RdVal, RsVal, finalResult, false)
		cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
	case 0x6: // SBC
		var carryIn uint64 = 0
		if (cpu.CPSR & BitC) == 0 {
			carryIn = 1
		}
		result := uint64(RdVal) - uint64(RsVal) - carryIn
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		carry := uint64(RdVal) >= uint64(RsVal)+carryIn
		overflow := CalculateOverflow(RdVal, RsVal, finalResult, true)
		cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
	case 0x7: // ROR
		RsVal &= 0xFF
		amount := RsVal & 0x1F
		result := (RdVal >> amount) | (RdVal << (32 - amount))
		cpu.WriteReg(Rd, result)
		carry := (cpu.CPSR & BitC) != 0
		if RsVal > 0 {
			if amount == 0 {
				carry = (RdVal & (1 << 31)) != 0
			} else {
				carry = (RdVal & (1 << (amount - 1))) != 0
			}
		}
		cpu.UpdateLogicalFlags(result, carry)
	case 0x8: // TST
		result := RdVal & RsVal
		cpu.UpdateLogicalFlags(result, false)
	case 0x9: // NEG
		result := -RsVal
		cpu.WriteReg(Rd, result)
		carry := RsVal == 0
		overflow := CalculateOverflow(0, RsVal, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	case 0xA: // CMP
		result := RdVal - RsVal
		carry := RdVal >= RsVal
		overflow := CalculateOverflow(RdVal, RsVal, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	case 0xB: // CMN
		result := uint64(RdVal) + uint64(RsVal)
		finalResult := uint32(result & 0xFFFFFFFF)
		carry := result > 0xFFFFFFFF
		overflow := CalculateOverflow(RdVal, RsVal, finalResult, false)
		cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
	case 0xC: // ORR
		result := RdVal | RsVal
		cpu.WriteReg(Rd, result)
		cpu.UpdateLogicalFlags(result, false)
	case 0xD: // MUL
		result := RdVal * RsVal
		cpu.WriteReg(Rd, result)
		cpu.UpdateLogicalFlags(result, false)
	case 0xE: // BIC
		result := RdVal & ^RsVal
		cpu.WriteReg(Rd, result)
		cpu.UpdateLogicalFlags(result, false)
	case 0xF: // MVN
		result := ^RsVal
		cpu.WriteReg(Rd, result)
		cpu.UpdateLogicalFlags(result, false)
	}
}

func (cpu *CPU) executeThumbMoveCompareAddSubtractImmediate(opcode uint16) {
	op := int((opcode >> 11) & 0x3)
	Rd := int((opcode >> 8) & 0x7)
	nn := uint32(opcode & 0xFF)
	RdVal := cpu.ReadReg(Rd)

	switch op {
	case 0x0: // MOV
		cpu.WriteReg(Rd, nn)
		cpu.UpdateLogicalFlags(nn, false)
	case 0x1: // CMP
		result := RdVal - nn
		carry := RdVal >= nn
		overflow := CalculateOverflow(RdVal, nn, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	case 0x2: // ADD
		result := uint64(RdVal) + uint64(nn)
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		carry := result > 0xFFFFFFFF
		overflow := CalculateOverflow(RdVal, nn, finalResult, false)
		cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
	case 0x3: // SUB
		result := RdVal - nn
		cpu.WriteReg(Rd, result)
		carry := RdVal >= nn
		overflow := CalculateOverflow(RdVal, nn, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	}
}

func (cpu *CPU) executeThumbAddSubtract(opcode uint16) {
	op := int((opcode >> 9) & 0x3)
	Rs := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)
	RsVal := cpu.ReadReg(Rs)

	switch op {
	case 0x0: // ADD Rd,Rs,Rn
		Rn := int((opcode >> 6) & 0x7)
		RnVal := cpu.ReadReg(Rn)
		result := uint64(RsVal) + uint64(RnVal)
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		carry := result > 0xFFFFFFFF
		overflow := CalculateOverflow(RsVal, RnVal, finalResult, false)
		cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
	case 0x1: // SUB Rd,Rs,Rn
		Rn := int((opcode >> 6) & 0x7)
		RnVal := cpu.ReadReg(Rn)
		result := RsVal - RnVal
		cpu.WriteReg(Rd, result)
		carry := RsVal >= RnVal
		overflow := CalculateOverflow(RsVal, RnVal, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	case 0x2: // ADD Rd,Rs,#nn
		nn := uint32((opcode >> 6) & 0x7)
		result := uint64(RsVal) + uint64(nn)
		finalResult := uint32(result & 0xFFFFFFFF)
		cpu.WriteReg(Rd, finalResult)
		carry := result > 0xFFFFFFFF
		overflow := CalculateOverflow(RsVal, nn, finalResult, false)
		cpu.UpdateArithmeticFlags(finalResult, carry, overflow)
	case 0x3: // SUB Rd,Rs,#nn
		nn := uint32((opcode >> 6) & 0x7)
		result := RsVal - nn
		cpu.WriteReg(Rd, result)
		carry := RsVal >= nn
		overflow := CalculateOverflow(RsVal, nn, result, true)
		cpu.UpdateArithmeticFlags(result, carry, overflow)
	}
}

func (cpu *CPU) executeThumbMoveShiftedRegister(opcode uint16) {
	op := int((opcode >> 11) & 0x3)
	offset := uint((opcode >> 6) & 0x1F)
	Rs := int((opcode >> 3) & 0x7)
	Rd := int(opcode & 0x7)
	RsVal := cpu.ReadReg(Rs)

	result, carry := Shift(RsVal, op, offset, false, false)
	cpu.WriteReg(Rd, result)
	cpu.UpdateLogicalFlags(result, carry)
}

func (cpu *CPU) ExecuteThumb(opcode uint16) {
	switch {
	case IsThumbSoftwareInterrupt(opcode):
		cpu.executeThumbSoftwareInterrupt()
	case IsThumbUnconditionalBranch(opcode):
		cpu.executeThumbUnconditionalBranch(opcode)
	case IsThumbConditionalBranch(opcode):
		cpu.executeThumbConditionalBranch(opcode)
	case IsThumbMultipleLoadStore(opcode):
		cpu.executeThumbMultipleLoadStore(opcode)
	case IsThumbLongBranchWithLink(opcode):
		cpu.executeThumbLongBranchWithLink(opcode)
	case IsThumbAddOffsetToStackPointer(opcode):
		cpu.executeThumbAddOffsetToStackPointer(opcode)
	case IsThumbPushPopRegisters(opcode):
		cpu.executeThumbPushPopRegisters(opcode)
	case IsThumbLoadStoreHalfword(opcode):
		cpu.executeThumbLoadStoreHalfword(opcode)
	case IsThumbSPRelativeLoadStore(opcode):
		cpu.executeThumbSPRelativeLoadStore(opcode)
	case IsThumbLoadAddress(opcode):
		cpu.executeThumbLoadAddress(opcode)
	case IsThumbLoadStoreWithImmediateOffset(opcode):
		cpu.executeThumbLoadStoreWithImmediateOffset(opcode)
	case IsThumbLoadStoreWithRegisterOffset(opcode):
		cpu.executeThumbLoadStoreWithRegisterOffset(opcode)
	case IsThumbLoadStoreSignExtendedByteHalfword(opcode):
		cpu.executeThumbLoadStoreSignExtendedByteHalfword(opcode)
	case IsThumbPCRelativeLoad(opcode):
		cpu.executeThumbPCRelativeLoad(opcode)
	case IsThumbHiRegisterOperationsBranchExchange(opcode):
		cpu.executeThumbHiRegisterOperationsBranchExchange(opcode)
	case IsThumbALUOperations(opcode):
		cpu.executeThumbALUOperations(opcode)
	case IsThumbMoveCompareAddSubtractImmediate(opcode):
		cpu.executeThumbMoveCompareAddSubtractImmediate(opcode)
	case IsThumbAddSubtract(opcode):
		cpu.executeThumbAddSubtract(opcode)
	case IsThumbMoveShiftedRegister(opcode):
		cpu.executeThumbMoveShiftedRegister(opcode)
	}
}
