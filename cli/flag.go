package cli

import (
	"flag"
	"fmt"
)

type Flag interface {
	fmt.Stringer
	Apply(set *flag.FlagSet)
	GetName() string
}


var HelpFlag Flag = BoolFlag{
	Name:  "help, h",
	Usage: "show help",
}

// FlagStringer converts a flag definition to a string. This is used by help
// to display a flag.
var FlagStringer FlagStringFunc = stringifyFlag

