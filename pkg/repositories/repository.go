package repositories

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/rs/zerolog/log"
)

// A repository is a collection of commands and aliases, that can optionally be reloaded
// through a watcher (and for which you can register callbacks, for example to update a potential
// cobra command or REST route).

type UpdateCallback func(cmd cmds.Command) error
type RemoveCallback func(cmd cmds.Command) error

type Repository struct {
	// The root of the repository.
	Root           *TrieNode
	Directories    []string
	updateCallback UpdateCallback
	removeCallback RemoveCallback

	// watching and loading commands
	loader     loaders.ReaderCommandLoader
	cmdOptions []cmds.CommandDescriptionOption
}

type RepositoryOption func(*Repository)

func WithDirectories(directories []string) RepositoryOption {
	return func(r *Repository) {
		r.Directories = directories
	}
}

func WithCommandLoader(loader loaders.ReaderCommandLoader) RepositoryOption {
	return func(r *Repository) {
		r.loader = loader
	}
}

func WithCommandDescriptionOptions(cmdOptions []cmds.CommandDescriptionOption) RepositoryOption {
	return func(r *Repository) {
		r.cmdOptions = cmdOptions
	}
}

func WithDirectory(directory string) RepositoryOption {
	return func(r *Repository) {
		r.Directories = append(r.Directories, directory)
	}
}

func WithUpdateCallback(callback UpdateCallback) RepositoryOption {
	return func(r *Repository) {
		r.updateCallback = callback
	}
}

func WithRemoveCallback(callback RemoveCallback) RepositoryOption {
	return func(r *Repository) {
		r.removeCallback = callback
	}
}

func WithCommands(commands ...cmds.Command) RepositoryOption {
	return func(r *Repository) {
		r.Add(commands...)
	}
}

// NewRepository creates a new repository.
func NewRepository(options ...RepositoryOption) *Repository {
	ret := &Repository{
		Root: NewTrieNode([]cmds.Command{}, []*alias.CommandAlias{}),
	}
	for _, opt := range options {
		opt(ret)
	}
	return ret
}

func (r *Repository) Add(commands ...cmds.Command) {
	aliases := []*alias.CommandAlias{}

	for _, command := range commands {
		_, isAlias := command.(*alias.CommandAlias)
		if isAlias {
			aliases = append(aliases, command.(*alias.CommandAlias))
			continue
		}

		prefix := command.Description().Parents
		r.Root.InsertCommand(prefix, command)
		if r.updateCallback != nil {
			err := r.updateCallback(command)
			if err != nil {
				log.Warn().Err(err).Msg("error while updating command")
			}
		}
	}

	for _, alias_ := range aliases {
		prefix := alias_.Parents
		aliasedCommand, ok := r.Root.FindCommand(prefix)
		if !ok {
			log.Warn().Msgf("alias_ %s for %s not found", alias_.Description().Name, alias_.AliasFor)
			continue
		}
		alias_.AliasedCommand = aliasedCommand

		r.Root.InsertCommand(prefix, alias_)
		if r.updateCallback != nil {
			err := r.updateCallback(alias_)
			if err != nil {
				log.Warn().Err(err).Msg("error while updating command")
			}
		}
	}
}

func (r *Repository) Remove(prefixes ...[]string) {
	for _, prefix := range prefixes {
		removedCommands := r.Root.Remove(prefix)
		for _, command := range removedCommands {
			if r.removeCallback != nil {
				err := r.removeCallback(command)
				if err != nil {
					log.Warn().Err(err).Msg("error while removing command")
				}
			}
		}
	}
}
