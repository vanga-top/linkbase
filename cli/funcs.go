package cli

type BashCompleteFunc func(ctx *Context)

type BeforeFunc func(*Context) error

type AfterFunc func(*Context) error

type CommandNotFoundFunc func(*Context, string)

type OnUsageErrorFunc func(context *Context, err error, isSubcommand bool) error

type FlagStringFunc func(Flag) string
