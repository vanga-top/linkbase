package master

import (
	"flag"
	"fmt"
	"os"
)

// ServerType constant
type ServerType int

const (
	MASTER ServerType = iota
	SLAVE_PROXY
	SLAVE_QUERY
)

type command interface {
	execute(args []string, flags *flag.FlagSet)
}

// RunLinkbaseMaster run linkbase master
func RunLinkbaseMaster(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, usageCommand)
		return
	}

}
