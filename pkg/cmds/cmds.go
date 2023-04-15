package cmds

import (
	"fmt"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"io/fs"
	"os"
)

type EmbeddedCommandLocation struct {
	FS      fs.FS
	Name    string
	Root    string
	DocRoot string
}

type CommandLocations struct {
	Embedded         []EmbeddedCommandLocation
	Repositories     []string
	AdditionalLayers []layers.ParameterLayer
	HelpSystem       *help.HelpSystem
}

type LoadCommandsOption func(*CommandLocations)

func NewCommandLocations(options ...LoadCommandsOption) *CommandLocations {
	ret := &CommandLocations{
		Embedded:         make([]EmbeddedCommandLocation, 0),
		Repositories:     make([]string, 0),
		AdditionalLayers: make([]layers.ParameterLayer, 0),
	}

	for _, o := range options {
		o(ret)
	}

	return ret
}

func WithEmbeddedLocations(locations ...EmbeddedCommandLocation) LoadCommandsOption {
	return func(c *CommandLocations) {
		c.Embedded = append(c.Embedded, locations...)
	}
}

func WithRepositories(locations ...string) LoadCommandsOption {
	return func(c *CommandLocations) {
		c.Repositories = append(c.Repositories, locations...)
	}
}

func WithAdditionalLayers(layers ...layers.ParameterLayer) LoadCommandsOption {
	return func(c *CommandLocations) {
		c.AdditionalLayers = append(c.AdditionalLayers, layers...)
	}
}

func WithHelpSystem(helpSystem *help.HelpSystem) LoadCommandsOption {
	return func(c *CommandLocations) {
		c.HelpSystem = helpSystem
	}
}

type CommandLoader[T glazed_cmds.Command] struct {
	locations *CommandLocations
}

func NewCommandLoader[T glazed_cmds.Command](locations *CommandLocations) *CommandLoader[T] {
	return &CommandLoader[T]{
		locations: locations,
	}
}

func (c *CommandLoader[T]) LoadCommands(
	loader loaders.FSCommandLoader,
	helpSystem *help.HelpSystem,
	options ...glazed_cmds.CommandDescriptionOption,
) ([]T, []*alias.CommandAlias, error) {
	// Load the variables from the environment

	log.Debug().
		Str("config", viper.ConfigFileUsed()).
		Msg("Loaded configuration")

	var commands []T
	var aliases []*alias.CommandAlias
	for _, e := range c.locations.Embedded {
		options_ := append([]glazed_cmds.CommandDescriptionOption{
			glazed_cmds.WithPrependSource("embed:" + e.Name + ":"),
			glazed_cmds.WithStripParentsPrefix([]string{e.Root}),
		}, options...)
		aliasOptions := []alias.Option{
			alias.WithPrependSource("embed:" + e.Name + ":"),
			alias.WithStripParentsPrefix([]string{e.Root}),
		}
		commands_, aliases_, err := loader.LoadCommandsFromFS(e.FS, e.Root, options_, aliasOptions)
		if err != nil {
			return nil, nil, err
		}
		for _, command := range commands_ {
			cmd, ok := command.(T)
			if !ok {
				return nil, nil, fmt.Errorf("command %s is not a GlazeCommand", command.Description().Name)
			}
			commands = append(commands, cmd)
		}
		aliases = append(aliases, aliases_...)

		err = helpSystem.LoadSectionsFromFS(e.FS, e.DocRoot)
		if err != nil {
			// if err is PathError, it means that the directory does not exist
			// and we can safely ignore it
			if _, ok := err.(*fs.PathError); !ok {
				return nil, nil, err
			}
		}

	}

	repositoryCommands, repositoryAliases, err := c.loadRepositoryCommands(loader, helpSystem, options...)
	if err != nil {
		return nil, nil, err
	}

	commands = append(commands, repositoryCommands...)
	aliases = append(aliases, repositoryAliases...)

	for _, command := range commands {
		description := command.Description()
		description.Layers = append(description.Layers, c.locations.AdditionalLayers...)
	}

	return commands, aliases, nil
}

func (c *CommandLoader[T]) loadRepositoryCommands(
	loader loaders.FSCommandLoader,
	helpSystem *help.HelpSystem,
	options ...glazed_cmds.CommandDescriptionOption,
) ([]T, []*alias.CommandAlias, error) {

	commands := make([]T, 0)
	aliases := make([]*alias.CommandAlias, 0)

	for _, repository := range c.locations.Repositories {
		repository = os.ExpandEnv(repository)

		// check that repository exists and is a directory
		s, err := os.Stat(repository)

		if os.IsNotExist(err) {
			log.Debug().Msgf("Repository %s does not exist", repository)
			continue
		} else if err != nil {
			log.Warn().Msgf("Error while checking directory %s: %s", repository, err)
			continue
		}

		if s == nil || !s.IsDir() {
			log.Warn().Msgf("Repository %s is not a directory", repository)
		} else {
			docDir := fmt.Sprintf("%s/doc", repository)
			options_ := append(options,
				glazed_cmds.WithPrependSource(repository+"/"),
				glazed_cmds.WithStripParentsPrefix([]string{"."}),
			)
			aliasOptions := []alias.Option{
				alias.WithPrependSource(repository + "/"),
			}
			commands_, aliases_, err := loader.LoadCommandsFromFS(
				os.DirFS(repository),
				".",
				options_,
				aliasOptions,
			)
			if err != nil {
				return nil, nil, err
			}

			for _, command := range commands_ {
				glazeCommand, ok := command.(T)
				if !ok {
					return nil, nil, fmt.Errorf("command %s is not a GlazeCommand", command.Description().Name)
				}
				commands = append(commands, glazeCommand)
			}
			aliases = append(aliases, aliases_...)

			_, err = os.Stat(docDir)
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				log.Debug().Err(err).Msgf("Error while checking directory %s", docDir)
				continue
			}
			err = helpSystem.LoadSectionsFromFS(os.DirFS(docDir), ".")
			if err != nil {
				log.Warn().Err(err).Msgf("Error while loading help sections from directory %s", repository)
				continue
			}
		}
	}
	return commands, aliases, nil
}
