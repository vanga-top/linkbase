package master

import (
	"flag"
	"fmt"
)

const CMD_RUN = "run"

type run struct {
}

func (r run) execute(args []string, flags *flag.FlagSet) {
	fmt.Println(args)
	fmt.Println(flags)
}
