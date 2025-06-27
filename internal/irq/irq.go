package irq

type IRQ struct {
	IME uint16
	IE  uint16
	IF  uint16
}

func NewIRQ() *IRQ {
	return &IRQ{}
}
