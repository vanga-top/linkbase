package cli

import "fmt"

var helpCommand = Command{
	Name:      "help",
	Aliases:   []string{"h"},
	Usage:     "show a list of commands or help for one command",
	ArgsUsage: "[command]",
	Action: func(c *Context) error {
		args := c.Args()
		if args.Present() {
			return ShowCommandHelp(c, args.First())
		}
		ShowAppHelp(c)
		return nil
	},
}

func ShowAppHelp(c *Context) {

}

func ShowCommandHelp(c *Context, first string) error {

	return nil
}

func DefaultAppComplete(c *Context) {
	for _, command := range c.App.Commands {
		if command.Hidden {
			continue
		}
		for _, name := range command.NamesWithHiddenAliases() {
			fmt.Fprintln(c.App.Writer, name)
		}
	}
}
