package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type Command struct {
	Name               string
	ShortName          string
	Aliases            []string
	Usage              string
	UsageText          string
	Description        string
	ArgsUsage          string
	Category           string
	BashComplete       BashCompleteFunc
	Before             BeforeFunc
	After              AfterFunc
	Action             interface{}
	OnUsageError       OnUsageErrorFunc
	Subcommands        Commands
	Flags              []Flag
	SkipFlagParsing    bool
	SkipArgReorder     bool
	HideHelp           bool
	HideHelpCommand    bool
	Hidden             bool
	HiddenAliases      bool
	HelpName           string
	commandNamePath    []string
	Prompt             string
	EnvVarSetCommand   string
	AssignmentOperator string
	DisableHistory     string
	EnableHistory      string
	CustomHelpTemplate string
}

// NamesWithHiddenAliases returns the names including short names and aliases.
func (c Command) NamesWithHiddenAliases() []string {
	names := []string{c.Name}

	if c.ShortName != "" {
		names = append(names, c.ShortName)
	}
	names = append(names, c.Aliases...)
	return names
}

func (c Command) HasName(name string) bool {
	for _, n := range c.NamesWithHiddenAliases() {
		if n == name {
			return true
		}
	}
	return false
}

// Commands is a slice of Command
type Commands []Command

type isBoolFlag interface {
	IsBoolFlag() bool
}

func (c Command) Run(ctx *Context) (err error) {
	if len(c.Subcommands) > 0 {
		return c.startApp(ctx)
	}

	if !c.HideHelp && (HelpFlag != BoolFlag{}) {
		// append help to flags
		c.Flags = append(
			c.Flags,
			HelpFlag,
		)
	}

	set, err := flagSet(c.Name, c.Flags)
	if err != nil {
		return err
	}
	set.SetOutput(io.Discard)

	if c.SkipFlagParsing {
		err = set.Parse(append([]string{"--"}, ctx.Args().Tail()...))
	} else if !c.SkipArgReorder {
		var (
			regularArgs, flagArgs []string
			terminatorIndex       = -1
			isFlagArg             bool
		)

		for index, arg := range ctx.Args().Tail() {
			doubleHyphen := false

			switch {
			case terminatorIndex > -1:
				regularArgs = append(regularArgs, arg)
			case isFlagArg:
				flagArgs = append(flagArgs, arg)
				isFlagArg = false
			case arg == "--":
				terminatorIndex = index
				regularArgs = append(regularArgs, arg)
			case arg == "-":
				regularArgs = append(regularArgs, arg)
			case strings.HasPrefix(arg, "--"):
				doubleHyphen = true
				fallthrough
			case strings.HasPrefix(arg, "-"):
				flagArgs = append(flagArgs, arg)
				if eq := strings.Index(arg, "="); eq > -1 {
					break
				}

				hyphens := "-"
				if doubleHyphen {
					hyphens += "-"
				}
				flagName := strings.TrimPrefix(arg, hyphens)
				f := set.Lookup(flagName)
				if f != nil {
					fv, ok := f.Value.(isBoolFlag)
					isFlagArg = !ok || !fv.IsBoolFlag()
				}
			default:
				regularArgs = append(regularArgs, arg)
			}
		}
		err = set.Parse(append(flagArgs, regularArgs...))
	} else {
		err = set.Parse(ctx.Args().Tail())
	}

	nerr := normalizeFlags(c.Flags, set)
	if nerr != nil {
		fmt.Fprintln(ctx.App.Writer, nerr)
		fmt.Fprintln(ctx.App.Writer)
		return nerr
	}

	context := NewContext(ctx.App, set, ctx)
	context.Command = c
	if checkCommandCompletions(context, c.Name) {
		return nil
	}

	if err != nil {
		if c.OnUsageError != nil {
			err := c.OnUsageError(context, err, false)
			HandleExitCoder(err)
			return err
		}
		fmt.Fprintln(context.App.Writer, "Incorrect Usage:", err.Error())
		fmt.Fprintln(context.App.Writer)
		return err
	}

	if checkCommandHelp(context, c.Name) {
		return nil
	}

	if c.After != nil {
		defer func() {
			afterErr := c.After(context)
			if afterErr != nil {
				HandleExitCoder(err)
				if err != nil {
					err = NewMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if c.Before != nil {
		err = c.Before(context)
		if err != nil {
			fmt.Fprintln(context.App.Writer, err)
			fmt.Fprintln(context.App.Writer)
			HandleExitCoder(err)
			return err
		}
	}

	if c.Action == nil {
		c.Action = helpSubcommand.Action
	}

	err = HandleAction(c.Action, context)

	if err != nil {
		HandleExitCoder(err)
	}
	return err
}

func (c Command) startApp(ctx *Context) error {
	app := NewApp()
	app.Metadata = ctx.App.Metadata
	// set the name and usage
	app.Name = fmt.Sprintf("%s %s", ctx.App.Name, c.Name)
	if c.HelpName == "" {
		app.HelpName = c.HelpName
	} else {
		app.HelpName = app.Name
	}

	app.Usage = c.Usage
	app.Description = c.Description
	app.ArgsUsage = c.ArgsUsage

	// set CommandNotFound
	app.CommandNotFound = ctx.App.CommandNotFound
	app.CustomAppHelpTemplate = c.CustomHelpTemplate

	// set the flags and commands
	app.Commands = c.Subcommands
	app.Flags = c.Flags
	app.HideHelp = c.HideHelp
	app.HideHelpCommand = c.HideHelpCommand

	app.Version = ctx.App.Version
	app.HideVersion = ctx.App.HideVersion
	app.Compiled = ctx.App.Compiled
	app.Author = ctx.App.Author
	app.Email = ctx.App.Email
	app.Writer = ctx.App.Writer
	app.HelpWriter = ctx.App.HelpWriter
	app.ErrWriter = ctx.App.ErrWriter

	app.categories = CommandCategories{}
	for _, command := range c.Subcommands {
		app.categories = app.categories.AddCommand(command.Category, command)
	}

	sort.Sort(app.categories)

	// bash completion
	app.EnableBashCompletion = ctx.App.EnableBashCompletion
	if c.BashComplete != nil {
		app.BashComplete = c.BashComplete
	}

	// set the actions
	app.Before = c.Before
	app.After = c.After
	if c.Action != nil {
		app.Action = c.Action
	} else {
		app.Action = helpSubcommand.Action
	}

	for index, cc := range app.Commands {
		app.Commands[index].commandNamePath = []string{c.Name, cc.Name}
	}

	return app.RunAsSubcommand(ctx)
}
