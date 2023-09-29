package cli

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
