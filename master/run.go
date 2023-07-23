package master

import (
	"flag"
	"fmt"
)

const RUM_CMD = "run"

type run struct {
}

func (r run) execute(args []string, flags *flag.FlagSet) {
	fmt.Println(args)
	fmt.Println(flags)
}
