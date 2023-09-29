package cli

type CommandCategories []*CommandCategory

type CommandCategory struct {
	Name     string
	Commands Commands
}
