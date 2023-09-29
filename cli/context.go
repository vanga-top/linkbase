package cli

import "flag"

type Context struct {
	App           *App
	Command       Command
	shellComplete bool
	flagSet       *flag.FlagSet
	setFlags      map[string]bool
	parentContext *Context
}

// Args contains apps console arguments
type Args []string

func (c *Context) Args() Args {
	args := Args(c.flagSet.Args())
	return args
}

// Present checks if there are any arguments present
func (a Args) Present() bool {
	return len(a) != 0
}

// Get returns the nth argument, or else a blank string
func (a Args) Get(n int) string {
	if len(a) > n {
		return a[n]
	}
	return ""
}

// First returns the first argument, or else a blank string
func (a Args) First() string {
	return a.Get(0)
}
