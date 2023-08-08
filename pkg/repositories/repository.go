package repositories

import "github.com/go-go-golems/glazed/pkg/cmds"

type Repository interface {
	CollectCommands(prefix []string, recurse bool) []cmds.Command
}
