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

// RunLinkbaseMaster main linkbase master
func RunLinkbaseMaster(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, usageCommand)
		return
	}
	cmd := args[1]
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, usageCommand)
	}

	var c command
	switch cmd {
	case CMD_RUN:
		c = &run{}
	case CMD_UPDATE:
		c = &update{}
	}
	c.execute(args, flags)
}
