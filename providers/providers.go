package providers

type Command string

const (
	CommandDump    Command = "dump"
	CommandRestore Command = "restore"
	CommandQuery   Command = "query"
)
