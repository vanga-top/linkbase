package cli

type Command struct {
	Name        string
	ShortName   string
	Aliases     []string
	Usage       string
	UsageText   string
	Description string
	ArgsUsage   string
	Category    string
}
