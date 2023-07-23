package master

import "fmt"

// [main/stop/update] [serverType]  -f -p -v
// main master -f
var (
	usageCommand = fmt.Sprintf("Usage:\n "+"%s\n%s\n%s\n", runCommand, serverType, flagCommand)

	runCommand = ``

	flagCommand = ``

	serverType = ``
)
