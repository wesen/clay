package fs

import (
	claycmds "github.com/go-go-golems/clay/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
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

	// fsLoader is used to load all commands on startup
	fsLoader loaders.FSCommandLoader
	// loader is used to reload a single command that changed while watching
	loader loaders.ReaderCommandLoader
	// these options are passed to the loader to create new descriptions
	cmdOptions []cmds.CommandDescriptionOption
}

type RepositoryOption func(*Repository)

func WithDirectories(directories []string) RepositoryOption {
	return func(r *Repository) {
		r.Directories = directories
	}
}

// WithCommandLoader sets the command loader to use when loading commands from
// an updated file when it changes.
func WithCommandLoader(loader loaders.ReaderCommandLoader) RepositoryOption {
	return func(r *Repository) {
		r.loader = loader
	}
}

// WithFSLoader sets the command loader to use when loading commands from
// the filesystem on startup or when a directory changes.
func WithFSLoader(loader loaders.FSCommandLoader) RepositoryOption {
	return func(r *Repository) {
		r.fsLoader = loader
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

// LoadCommands initializes the repository by loading all commands from the loader,
// if available.
func (r *Repository) LoadCommands() error {
	if r.fsLoader != nil {
		// TODO(manuel, 2023-05-26): Expose the repositories helpsystem
		// We currently do not provide or use the helpsystem,
		// but see:
		// https://github.com/go-go-golems/glazed/issues/163
		helpSystem := help.NewHelpSystem()
		locations := claycmds.CommandLocations{
			Repositories: r.Directories,
		}
		commandLoader := claycmds.NewCommandLoader[cmds.Command](&locations)
		commands, aliases, err := commandLoader.LoadCommands(r.fsLoader, helpSystem)
		if err != nil {
			return err
		}
		r.Add(commands...)
		for _, alias := range aliases {
			r.Add(alias)
		}
	}

	return nil
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

func (r *Repository) CollectCommands(prefix []string, recurse bool) []cmds.Command {
	return r.Root.CollectCommands(prefix, recurse)
}
