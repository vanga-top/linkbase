package master

import "fmt"

// [run/stop/update] [serverType]  -f -p -v
// run master -f
var (
	usageCommand = fmt.Sprintf("Usage:\n "+"%s\n%s\n%s\n", runCommand, serverType, flagCommand)

	runCommand = ``

	flagCommand = ``

	serverType = ``
)
