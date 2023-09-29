package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type App struct {
	Name string
	// Full name of command for help, defaults to Name
	HelpName             string
	Usage                string
	UsageText            string
	ArgsUsage            string
	Version              string
	Description          string
	Commands             []Command
	Flags                []Flag
	EnableBashCompletion bool
	HideHelp             bool
	HideHelpCommand      bool
	HideVersion          bool
	categories           CommandCategories
	BashComplete         BashCompleteFunc
	Before               BeforeFunc
	After                AfterFunc
	Action               interface{}
	CommandNotFound      CommandNotFoundFunc
	OnUsageError         OnUsageErrorFunc
	//compilation time
	Compiled              time.Time
	Authors               []Author
	Copyright             string
	Author                string
	Email                 string
	Writer                io.Writer
	HelpWriter            io.Writer
	ErrWriter             io.Writer
	Metadata              map[string]interface{}
	ExtraInfo             func() map[string]string
	CustomAppHelpTemplate string
	didSetup              bool
}

// Author represents someone who has contributed to a cli project.
type Author struct {
	Name  string // The Authors name
	Email string // The Authors email
}

func NewApp() *App {
	return &App{
		Name:         filepath.Base(os.Args[0]),
		HelpName:     filepath.Base(os.Args[0]),
		Usage:        "A New Cli Application",
		UsageText:    "",
		Version:      "0.0.0",
		BashComplete: DefaultAppComplete,
		Action:       helpCommand.Action,
		Compiled:     compileTime(),
		Writer:       os.Stdout,
		HelpWriter:   os.Stdout,
	}
}

func (a *App) Setup() {
	if a.didSetup {
		return
	}

	defer func() {
		a.didSetup = true
	}()

	if a.Author != "" || a.Email != "" {
		a.Authors = append(a.Authors, Author{Name: a.Author, Email: a.Email})
	}

	newCmds := []Command{}
	for _, c := range a.Commands {
		if c.HelpName == "" {
			c.HelpName = fmt.Sprintf("%s  %s", a.HelpName, c.Name)
		}
		newCmds = append(newCmds, c)
	}
	a.Commands = newCmds

	if a.Command(helpCommand.Name) == nil {
		if !a.HideHelpCommand {
			a.Commands = append(a.Commands,helpCommand)
		}
		if !a.HideHelp && (helpF) {
			
		}
	}
}

func (a *App) Command(name string) *Command {
	for _, c := range a.Commands {
		if c.HasName(name) {
			return &c
		}
	}
	return nil
}

func compileTime() time.Time {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}
