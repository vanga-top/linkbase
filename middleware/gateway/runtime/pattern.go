package runtime

import "github.com/linkbase/utils"

type op struct {
	code    utils.OpCode
	operand int
}

type Pattern struct {
	ops       []op
	pool      []string
	vars      []string
	stackSize int
	tailLen   int
	verb      string
}
