package interrupt

type InterruptController struct {
	IME uint32
	IE  uint16
	IF  uint16
}

func NewInterruptController() *InterruptController {
	return &InterruptController{}
}
